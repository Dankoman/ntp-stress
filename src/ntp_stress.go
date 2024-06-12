package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beevik/ntp"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

type Stats struct {
	TotalRequests  int
	FailedRequests int
	mu             sync.Mutex
}

func sendNTPRequest(server string, stats *Stats, wg *sync.WaitGroup) {
	defer wg.Done()
	_, err := ntp.Time(server)
	stats.mu.Lock()
	stats.TotalRequests++
	if err != nil {
		stats.FailedRequests++
	}
	stats.mu.Unlock()
}

func stressTestNTP(server string, startRate, maxRate, increment int, duration time.Duration) ([]int, []float64) {
	stats := &Stats{}
	var failedRequests []int
	var failPercentages []float64

	for rate := startRate; rate <= maxRate; rate += increment {
		startTime := time.Now()
		var wg sync.WaitGroup
		iterationStats := &Stats{}
		for i := 0; i < rate; i++ {
			wg.Add(1)
			go sendNTPRequest(server, iterationStats, &wg)
		}
		wg.Wait()
		elapsedTime := time.Since(startTime)
		failPercentage := float64(iterationStats.FailedRequests) / float64(iterationStats.TotalRequests) * 100
		fmt.Printf("Rate: %d requests/sec - Total: %d - Failed: %d - Fail %%: %.2f\n", rate, iterationStats.TotalRequests, iterationStats.FailedRequests, failPercentage)
		failedRequests = append(failedRequests, iterationStats.FailedRequests)
		failPercentages = append(failPercentages, failPercentage)
		time.Sleep(duration - elapsedTime)

		stats.mu.Lock()
		stats.TotalRequests += iterationStats.TotalRequests
		stats.FailedRequests += iterationStats.FailedRequests
		stats.mu.Unlock()
	}
	finalFailPercentage := float64(stats.FailedRequests) / float64(stats.TotalRequests) * 100
	fmt.Printf("Final Statistics - Total Requests: %d - Failed Requests: %d - Fail %%: %.2f\n", stats.TotalRequests, stats.FailedRequests, finalFailPercentage)

	return failedRequests, failPercentages
}

func plotResults(failedRequests []int, failPercentages []float64, startRate, maxRate, increment int) {
	p := plot.New()

	p.Title.Text = "NTP Server Stress Test"
	p.X.Label.Text = "Request Rate (requests/sec)"
	p.Y.Label.Text = "Failed Requests"

	pts := make(plotter.XYs, len(failedRequests))
	for i := range pts {
		pts[i].X = float64(startRate + i*increment)
		pts[i].Y = float64(failedRequests[i])
	}

	err := plotutil.AddLinePoints(p, "Failed Requests", pts)
	if err != nil {
		panic(err)
	}

	if err := p.Save(10*vg.Inch, 5*vg.Inch, "ntp_stress_test.png"); err != nil {
		panic(err)
	}

	pFail := plot.New()

	pFail.Title.Text = "NTP Server Stress Test Fail Percentages"
	pFail.X.Label.Text = "Request Rate (requests/sec)"
	pFail.Y.Label.Text = "Fail Percentage"

	ptsFail := make(plotter.XYs, len(failPercentages))
	for i := range ptsFail {
		ptsFail[i].X = float64(startRate + i*increment)
		ptsFail[i].Y = failPercentages[i]
	}

	err = plotutil.AddLinePoints(pFail, "Fail Percentage", ptsFail)
	if err != nil {
		panic(err)
	}

	if err := pFail.Save(10*vg.Inch, 5*vg.Inch, "ntp_stress_test_fail_percentage.png"); err != nil {
		panic(err)
	}
}

func getInput(scanner *bufio.Scanner, prompt string, defaultValue string) string {
	fmt.Print(prompt)
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			return defaultValue
		}
		return input
	}
	return defaultValue
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("NTP Server Stress Test")

	for {
		server := getInput(scanner, "Enter the NTP server (default: pool.ntp.org): ", "pool.ntp.org")

		startRateStr := getInput(scanner, "Enter the starting request rate (requests per second): ", "1")
		startRate, err := strconv.Atoi(startRateStr)
		if err != nil {
			fmt.Println("Invalid input. Using default start rate of 1.")
			startRate = 1
		}

		maxRateStr := getInput(scanner, "Enter the maximum request rate (requests per second): ", "1000")
		maxRate, err := strconv.Atoi(maxRateStr)
		if err != nil {
			fmt.Println("Invalid input. Using default max rate of 1000.")
			maxRate = 1000
		}

		incrementStr := getInput(scanner, "Enter the increment rate (requests per second): ", "1")
		increment, err := strconv.Atoi(incrementStr)
		if err != nil {
			fmt.Println("Invalid input. Using default increment of 1.")
			increment = 1
		}

		durationStr := getInput(scanner, "Enter the duration for each increment (seconds): ", "1")
		durationSec, err := strconv.Atoi(durationStr)
		if err != nil {
			fmt.Println("Invalid input. Using default duration of 1 second.")
			durationSec = 1
		}
		duration := time.Duration(durationSec) * time.Second

		// Calculate the total duration of the test
		totalSteps := (maxRate - startRate) / increment
		totalDuration := time.Duration(totalSteps) * duration

		fmt.Printf("\nTest configuration:\n")
		fmt.Printf("NTP server: %s\n", server)
		fmt.Printf("Starting request rate: %d requests/sec\n", startRate)
		fmt.Printf("Maximum request rate: %d requests/sec\n", maxRate)
		fmt.Printf("Increment rate: %d requests/sec\n", increment)
		fmt.Printf("Duration per increment: %d seconds\n", durationSec)
		fmt.Printf("Estimated total test duration: %s\n", totalDuration)

		proceed := getInput(scanner, "Do you want to proceed with these settings? (yes/no): ", "no")

		if strings.ToLower(proceed) == "yes" {
			failedRequests, failPercentages := stressTestNTP(server, startRate, maxRate, increment, duration)
			plotResults(failedRequests, failPercentages, startRate, maxRate, increment)
			break
		} else {
			fmt.Println("Let's try setting the parameters again.")
		}
	}
}
