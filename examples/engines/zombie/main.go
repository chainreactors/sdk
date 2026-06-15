package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/chainreactors/sdk/pkg/types"
	"github.com/chainreactors/sdk/zombie"
)

var (
	target   = flag.String("target", "", "Target IP (required)")
	service  = flag.String("service", "ssh", "Service name (ssh, mysql, redis, ftp, smb, ...)")
	port     = flag.String("port", "", "Port (default: service default port)")
	mode     = flag.String("mode", "brute", "Attack mode: brute, pitchfork, sniper")
	users    = flag.String("users", "root,admin", "Usernames (comma-separated)")
	passwords = flag.String("passwords", "123456,admin,password", "Passwords (comma-separated)")
	auths    = flag.String("auths", "", "Pitchfork auth pairs (user::pass,user2::pass2)")
	threads  = flag.Int("threads", 10, "Number of threads")
	timeout  = flag.Int("timeout", 5, "Timeout in seconds")
)

func main() {
	flag.Parse()

	if *target == "" {
		fmt.Fprintln(os.Stderr, "Usage: zombie -target <ip> -service <name> [-port <port>] [-mode brute|pitchfork|sniper]")
		fmt.Fprintln(os.Stderr, "\nBrute (cartesian product):")
		fmt.Fprintln(os.Stderr, "  zombie -target 192.168.1.1 -service ssh -users root,admin -passwords 123456,admin")
		fmt.Fprintln(os.Stderr, "\nPitchfork (paired credentials):")
		fmt.Fprintln(os.Stderr, "  zombie -target 192.168.1.1 -service mysql -mode pitchfork -auths root::123456,admin::admin")
		fmt.Fprintln(os.Stderr, "\nSniper (one attempt per target with its own credentials):")
		fmt.Fprintln(os.Stderr, "  zombie -target 192.168.1.1 -service redis -port 6379 -mode sniper -users root -passwords 123456")
		os.Exit(1)
	}

	engine, err := zombie.NewEngine(zombie.NewConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating engine: %v\n", err)
		os.Exit(1)
	}

	ctx := zombie.NewContext().
		SetThreads(*threads).
		SetTimeout(*timeout)

	targets := []zombie.Target{
		{IP: *target, Service: *service, Port: *port},
	}

	switch *mode {
	case "brute":
		userList := strings.Split(*users, ",")
		passList := strings.Split(*passwords, ",")
		results, err := engine.Brute(ctx, targets, userList, passList)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Brute failed: %v\n", err)
			os.Exit(1)
		}
		printResults(results)

	case "pitchfork":
		if *auths == "" {
			fmt.Fprintln(os.Stderr, "Error: -auths is required for pitchfork mode")
			os.Exit(1)
		}
		var authList []zombie.Auth
		for _, pair := range strings.Split(*auths, ",") {
			parts := strings.SplitN(pair, "::", 2)
			if len(parts) != 2 {
				fmt.Fprintf(os.Stderr, "Invalid auth pair: %s (expected user::pass)\n", pair)
				os.Exit(1)
			}
			authList = append(authList, zombie.Auth{Username: parts[0], Password: parts[1]})
		}
		results, err := engine.Pitchfork(ctx, targets, authList)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Pitchfork failed: %v\n", err)
			os.Exit(1)
		}
		printResults(results)

	case "sniper":
		targets[0].Username = strings.Split(*users, ",")[0]
		targets[0].Password = strings.Split(*passwords, ",")[0]
		results, err := engine.Sniper(ctx, targets)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Sniper failed: %v\n", err)
			os.Exit(1)
		}
		printResults(results)

	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s\n", *mode)
		os.Exit(1)
	}
}

func printResults(results []*types.ZombieResult) {
	if len(results) == 0 {
		fmt.Println("No results")
		return
	}
	for _, r := range results {
		fmt.Printf("%s://%s:%s  %s:%s\n",
			r.Service, r.IP, r.Port, r.Username, r.Password)
	}
	fmt.Printf("\nTotal: %d result(s)\n", len(results))
}
