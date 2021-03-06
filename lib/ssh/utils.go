/*
Open Source Initiative OSI - The MIT License (MIT):Licensing
The MIT License (MIT)
Copyright (c) 2017 Ralph Caraveo (deckarep@gmail.com)
Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
of the Software, and to permit persons to whom the Software is furnished to do
so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package ssh

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/user"
	"path"
	"strings"
	"sync"

	"github.com/deckarep/blade/lib/recipe"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/gobwas/glob"
	"github.com/mikkeloscar/sshconfig"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type hostGlobItem struct {
	glob  glob.Glob
	entry *sshconfig.SSHHost
}

const (
	defaultUnmatchedUser = "root"
)

var (
	globMatcher  glob.Glob
	hostGlobList []*hostGlobItem
)

func init() {
	hostGlobList = createHostGlobList()
}

func lookupUsernameForHost(host string) string {
	// TODO: clean this up but we have to strip the :22 port.
	actualHost := strings.Replace(host, ":22", "", -1)

	// Precedence of username returned:
	// 	1. First host glob match.
	// 	2. Full * (wildcard) for host glob.
	// 	3. HOME user if able to get.
	// 	4. default user to fallback on.
	for _, hostGlob := range hostGlobList {
		if hostGlob.glob.Match(actualHost) {
			return hostGlob.entry.User
		}
	}

	user, err := user.Current()
	if err != nil {
		log.Printf("%s: Couldn't get username for local user\n", color.YellowString("WARN"))
		return defaultUnmatchedUser
	}
	return user.Username
}

func createHostGlobList() []*hostGlobItem {
	var list []*hostGlobItem

	mapping, err := parseSSHConfig()
	if err != nil {
		log.Println("Failed to parse SSHConfig mapping")
	}

	for _, item := range mapping {
		for _, h := range item.Host {
			hostGlob, err := glob.Compile(h)
			if err != nil {
				log.Fatalf("%s: Host glob: %s not in a valid glob format within ~/.ssh/config file\n", color.RedString("ERROR"), h)
			}
			list = append(list, &hostGlobItem{
				glob:  hostGlob,
				entry: item,
			})
		}
	}

	return list
}

// SSHAgent queries the host operating systems SSH agent.
func SSHAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}
	return nil
}

func parseSSHConfig() ([]*sshconfig.SSHHost, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}

	hosts, err := sshconfig.ParseSSHConfig(path.Join(usr.HomeDir, ".ssh", "config"))
	if err != nil {
		return nil, err
	}

	return hosts, nil
}

func consumeAndLimitConcurrency(recipe *recipe.BladeRecipeYaml, commands []string, concurrency int) {
	// Limit the amount of concurrent ssh sessions.
	concurrencySem := make(chan int, concurrency)

	for host := range hostQueue {
		concurrencySem <- 1
		go func(h string) {
			defer func() {
				<-concurrencySem
				hostWg.Done()
			}()
			executeSession(recipe, h, commands)
		}(host)
	}
}

func enqueueHost(host string, port int) {
	host = strings.TrimSpace(host)

	// If it doesn't contain the port; add it.
	if !strings.Contains(host, ":") {
		host = fmt.Sprintf("%s:%d", host, port)
	}

	// Ignore what can't be parsed as host:port.
	_, _, err := net.SplitHostPort(host)
	if err != nil {
		log.Printf("Couldn't parse: %s", host)
		return
	}

	// Finally, enqueue it up for processing.
	hostQueue <- host
	hostWg.Add(1)
}

func consumeReaderPipes(wg *sync.WaitGroup, host string, rdr io.Reader, isStdErr bool, attempt int) {
	defer wg.Done()

	logHost := color.CyanString(host + ":")

	if isStdErr {
		attemptString := ""
		if attempt > 1 {
			attemptString = fmt.Sprintf(" (%s attempt)", humanize.Ordinal(attempt))
		}
		logHost = color.RedString(host + attemptString + ":")
	}

	scanner := bufio.NewScanner(rdr)
	for scanner.Scan() {
		sessionLogger.Println(logHost + " " + scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		sessionLogger.Print(color.RedString(host) + ": Error reading output from this host.")
	}
}
