package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

type (
	// Config holds input parameters for the plugin
	Config struct {
		InitOptions      InitOptions
		Cacert           string
		Sensitive        bool
		RoleARN          string
		RootDir          string
		TerraformDataDir string
		ExportEnvs       bool
		EnvFile          string
		EnvPrefix        string
	}

	// Netrc is credentials for cloning
	Netrc struct {
		Machine  string
		Login    string
		Password string
	}

	// InitOptions include options for the Terraform's init command
	InitOptions struct {
		BackendConfig []string `json:"backend-config"`
		Lock          *bool    `json:"lock"`
		LockTimeout   string   `json:"lock-timeout"`
	}

	// Plugin represents the plugin instance to be executed
	Plugin struct {
		Config    Config
		Netrc     Netrc
		Terraform Terraform
	}

	TerraformOutput struct {
		Sensitive bool        `json:"sensitive"`
		Type      string      `json:"type"`
		Value     interface{} `json:"value"`
	}
)

// Exec executes the plugin
func (p Plugin) Exec() error {
	// Install specified version of terraform
	if p.Terraform.Version != "" {
		err := installTerraform(p.Terraform.Version)

		if err != nil {
			return err
		}
	}

	if p.Config.RoleARN != "" {
		assumeRole(p.Config.RoleARN)
	}

	// writing the .netrc file with Github credentials in it.
	err := writeNetrc(p.Netrc.Machine, p.Netrc.Login, p.Netrc.Password)
	if err != nil {
		return err
	}

	var terraformDataDir string = ".terraform"
	if p.Config.TerraformDataDir != "" {
		terraformDataDir = p.Config.TerraformDataDir
		os.Setenv("TF_DATA_DIR", p.Config.TerraformDataDir)
	}

	var commands []*exec.Cmd

	commands = append(commands, exec.Command("terraform", "version"))

	if p.Config.Cacert != "" {
		commands = append(commands, installCaCert(p.Config.Cacert))
	}

	commands = append(commands, deleteCache(terraformDataDir))
	commands = append(commands, initCommand(p.Config.InitOptions))
	commands = append(commands, getModules())

	for _, c := range commands {
		if c.Dir == "" {
			wd, err := os.Getwd()
			if err == nil {
				c.Dir = wd
			}
		}
		if p.Config.RootDir != "" {
			c.Dir = c.Dir + "/" + p.Config.RootDir
		}
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if !p.Config.Sensitive {
			trace(c)
		}

		err := c.Run()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Fatal("Failed to execute a command")
		}
		logrus.Debug("Command completed successfully")
	}

	err = tfOutput(p.Config)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Failed to execute a command")
	}
	return nil
}

func assumeRole(roleArn string) {
	client := sts.New(session.New())
	duration := time.Hour * 1
	stsProvider := &stscreds.AssumeRoleProvider{
		Client:          client,
		Duration:        duration,
		RoleARN:         roleArn,
		RoleSessionName: "drone",
	}

	value, err := credentials.NewCredentials(stsProvider).Get()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Error assuming role!")
	}
	os.Setenv("AWS_ACCESS_KEY_ID", value.AccessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", value.SecretAccessKey)
	os.Setenv("AWS_SESSION_TOKEN", value.SessionToken)
}

func deleteCache(terraformDataDir string) *exec.Cmd {
	return exec.Command(
		"rm",
		"-rf",
		terraformDataDir,
	)
}

func getModules() *exec.Cmd {
	return exec.Command(
		"terraform",
		"get",
	)
}

func initCommand(config InitOptions) *exec.Cmd {
	args := []string{
		"init",
	}

	for _, v := range config.BackendConfig {
		args = append(args, fmt.Sprintf("-backend-config=%s", v))
	}

	// True is default in TF
	if config.Lock != nil {
		args = append(args, fmt.Sprintf("-lock=%t", *config.Lock))
	}

	// "0s" is default in TF
	if config.LockTimeout != "" {
		args = append(args, fmt.Sprintf("-lock-timeout=%s", config.LockTimeout))
	}

	// Fail Terraform execution on prompt
	args = append(args, "-input=false")

	return exec.Command(
		"terraform",
		args...,
	)
}

func installCaCert(cacert string) *exec.Cmd {
	ioutil.WriteFile("/usr/local/share/ca-certificates/ca_cert.crt", []byte(cacert), 0644)
	return exec.Command(
		"update-ca-certificates",
	)
}

func trace(cmd *exec.Cmd) {
	fmt.Println("$", strings.Join(cmd.Args, " "))
}

func tfOutput(config Config) error {
	c := exec.Command(
		"terraform",
		"output",
		"-json",
		"-no-color",
	)
	if c.Dir == "" {
		wd, err := os.Getwd()
		if err == nil {
			c.Dir = wd
		}
	}
	if config.RootDir != "" {
		c.Dir = c.Dir + "/" + config.RootDir
	}
	out, err := c.Output()
	if err != nil {
		return err
	}

	err = processOutput(config, out)
	if err != nil {
		return err
	}

	return nil
}

func processOutput(config Config, out []byte) error {
	var j map[string]TerraformOutput
	var b []byte

	json.Unmarshal(out, &j)

	fmt.Println("Outputs:")

	format := "%s%s%s=%s\n"
	sexp := ""
	if config.ExportEnvs {
		sexp = "export "
	}

	for k, v := range j {
		str := fmt.Sprintf(format, sexp, config.EnvPrefix, k, v.Value)
		if !v.Sensitive {
			fmt.Printf(format, sexp, config.EnvPrefix, k, v.Value)
		} else {
			fmt.Printf(format, sexp, config.EnvPrefix, k, "XXXXXXX")
		}
		b = append(b, str...)
	}

	return ioutil.WriteFile(config.EnvFile, b, 0644)
}

// helper function to write a netrc file.
// The following code comes from the official Git plugin for Drone:
// https://github.com/drone-plugins/drone-git/blob/8386effd2fe8c8695cf979427f8e1762bd805192/utils.go#L43-L68
func writeNetrc(machine, login, password string) error {
	if machine == "" {
		return nil
	}
	out := fmt.Sprintf(
		netrcFile,
		machine,
		login,
		password,
	)

	home := "/root"
	u, err := user.Current()
	if err == nil {
		home = u.HomeDir
	}
	path := filepath.Join(home, ".netrc")
	return ioutil.WriteFile(path, []byte(out), 0600)
}

const netrcFile = `
machine %s
login %s
password %s
`
