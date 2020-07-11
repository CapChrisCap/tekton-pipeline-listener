package main

import (
	"bufio"
	"os"
	"os/exec"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/common/log"
)

var VALID_PIPELINE = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

func main() {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": 1,
		})
	})

	r.Use(gin.BasicAuth(*getBasicAccounts()))

	r.GET("/logs/:pipelineRun", func(c *gin.Context) {
		pipelineRun := c.Param("pipelineRun")
		if !VALID_PIPELINE.MatchString(pipelineRun) {
			c.JSON(400, gin.H{
				"error": "Invalid pipeline RunID",
			})
			return
		}

		cmd := exec.Command("tkn", "pipelineruns", "logs", "-n", "tekton-pipelines", pipelineRun)

		cmd.Env = append(os.Environ(),
			"FORCE_COLOR=true",
		)
		// cmd.Stdout = os.Stdout
		stdout, _ := cmd.StdoutPipe()
		scanner := bufio.NewScanner(stdout)
		scanner.Split(bufio.ScanLines)

		go func() {
			for scanner.Scan() {
				c.Writer.WriteString(scanner.Text() + "\n")
			}
		}()
		cmd.Start()
		cmd.Wait()
	})

	if err := r.Run(":" + getPort()); err != nil {
		log.Fatalf("Web server could not start: %s", err.Error())
	}
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return port
}

func getBasicAccounts() *gin.Accounts {
	username := os.Getenv("BASIC_USERNAME")
	password := os.Getenv("BASIC_PASSWORD")

	if username == "" {
		username = "admin"
	}
	if password == "" {
		password = "admin"
	}

	return &gin.Accounts{
		username: password,
	}
}
