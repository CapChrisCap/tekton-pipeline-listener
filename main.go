package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"regexp"

	"github.com/gin-gonic/gin"
)

var VALID_PIPELINE = regexp.MustCompile(`^[a-z0-9-_]+$`)

type PipelineRun struct {
	Status struct {
		Conditions []struct {
			Message string `json:"message"`
			Reason  string `json:"reason"`
			Type    string `json:"type"`
			Status  string `json:"status"`
		} `json:"conditions"`
	} `json:"status"`
}

func main() {
	r := gin.Default()
	r.GET("/health", healthHandler)

	r.Use(gin.BasicAuth(*getBasicAccounts()))
	r.GET("/logs/:namespace/:pipelineRun", logsHandler)
	r.GET("/status/:namespace/:pipelineRun", statusHandler)

	if err := r.Run(":" + getPort()); err != nil {
		log.Fatalf("Web server could not start: %s", err.Error())
	}
}

func healthHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": 1,
	})
}

func logsHandler(c *gin.Context) {
	namespace := c.Param("namespace")
	pipelineRun := c.Param("pipelineRun")
	if !VALID_PIPELINE.MatchString(pipelineRun) {
		c.JSON(400, gin.H{
			"error": "Invalid pipeline RunID",
		})
		return
	}

	cmd := exec.Command("tkn", "pipelineruns", "logs", "-f", "-n", namespace, pipelineRun)

	cmd.Env = append(os.Environ(),
		"FORCE_COLOR=true",
	)
	// cmd.Stdout = os.Stdout
	stdout, _ := cmd.StdoutPipe()
	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanLines)

	go func() {
		for scanner.Scan() {
			_, scanErr := c.Writer.WriteString(scanner.Text() + "\n")
			if scanErr != nil {
				log.Printf("An error while scanning occured: %v", scanErr)
			}
		}
	}()
	err := cmd.Run()
	if err != nil {
		c.JSON(404, gin.H{
			"error": "pipeline not found",
		})
	}
}

func statusHandler(c *gin.Context) {
	namespace := c.Param("namespace")
	pipelineRun := c.Param("pipelineRun")
	if !VALID_PIPELINE.MatchString(pipelineRun) {
		c.JSON(400, gin.H{
			"error": "Invalid pipeline RunID",
		})
		return
	}

	cmd := exec.Command("tkn", "pipelineruns", "describe", "-o", "json", "-n", namespace, pipelineRun)
	cmd.Env = append(os.Environ(), "FORCE_COLOR=true")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		c.JSON(404, gin.H{
			"error": "pipeline not found",
		})
		return
	}

	pipelineRunDef := out.String()
	fullPipelineRun := PipelineRun{}
	err = json.Unmarshal([]byte(pipelineRunDef), &fullPipelineRun)
	if err != nil {
		log.Fatalf("Can not parse pipelinerun: %v", err)
	}

	conditions := fullPipelineRun.Status.Conditions
	lastState := conditions[len(conditions)-1]

	statusCode := 400
	output := lastState
	if lastState.Status == "True" {
		statusCode = 200
	}
	c.JSON(statusCode, output)
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return port
}

func getBasicAccounts() *gin.Accounts {
	username := os.Getenv("AUTH_USERNAME")
	password := os.Getenv("AUTH_PASSWORD")

	if username == "" {
		log.Fatalf("No AUTH_USERNAME environment variable has been provided for basic authentication")
	}
	if password == "" {
		log.Fatalf("No AUTH_PASSWORD environment variable has been provided for basic authentication")
	}

	return &gin.Accounts{
		username: password,
	}
}
