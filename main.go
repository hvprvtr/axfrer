package main

import (
	"bufio"
	"fmt"
	"github.com/fatih/color"
	"github.com/miekg/dns"
	"net"
	"os"
	"strings"
	"sync"
)

func main() {
	threads_limit := 10

	if len(os.Args) < 2 {
		fmt.Printf("USAGE: %s DOMAINS_LIST\n", os.Args[0])
		os.Exit(1)
	}

	var waitGroup sync.WaitGroup
	waitGroup.Add(threads_limit)
	input := make(chan string, threads_limit)

	file, err := os.Open(os.Args[1])
	if err != nil {
		println("File open error!")
		panic(err)
	}
	defer file.Close()

	for i := 0; i < threads_limit; i++ {
		go worker(input, &waitGroup)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		input <- strings.Trim(scanner.Text(), "\n\t\r ")
	}

	close(input)
	waitGroup.Wait()
}

func worker(input chan string, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()

	for domain := range input {
		isAxfrPossible(domain)
	}
}

func isAxfrPossible(domain string) {
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	srvs := []string{}

	nameserver, err := net.LookupNS(domain)
	if err != nil {
		fmt.Printf("%s Domain %s => %s\n", red("[ERROR]"), domain, err)
		return
	}

	for _, ns := range nameserver {
		srv := ns.Host[:len(ns.Host)-1]
		srvs = append(srvs, srv)
	}

	if len(srvs) == 0 {
		fmt.Printf("%s Domain %s has no NS!\n", red("[ERROR]"), domain)
		return
	}

	for _, ns := range srvs {
		tr := dns.Transfer{}
		m := &dns.Msg{}
		m.SetAxfr(dns.Fqdn(domain + "."))
		in, err := tr.In(m, ns+":53")
		if err != nil {
			continue
		}

		cnt := 0
		for ex := range in {
			for _, _ = range ex.RR {
				cnt++
			}
		}

		if cnt > 0 {
			fmt.Printf("%s @%s: %s\n", green("[GOOD]"), ns, domain)
		} else {
			fmt.Printf("%s @%s: %s\n", yellow("[SAD]"), ns, domain)
		}
	}
}
