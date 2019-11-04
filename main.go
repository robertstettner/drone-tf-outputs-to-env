package main

import (
	"encoding/json"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
)

var revision string // build number set at compile-time

func main() {
	app := cli.NewApp()
	app.Name = "terraform plugin"
	app.Usage = "terraform plugin"
	app.Action = run
	app.Version = revision
	app.Flags = []cli.Flag{

		//
		// plugin args
		//

		cli.StringFlag{
			Name:   "ca_cert",
			Usage:  "ca cert to add to your environment to allow terraform to use internal/private resources",
			EnvVar: "PLUGIN_CA_CERT",
		},
		cli.StringFlag{
			Name:   "init_options",
			Usage:  "options for the init command. See https://www.terraform.io/docs/commands/init.html",
			EnvVar: "PLUGIN_INIT_OPTIONS",
		},
		cli.StringFlag{
			Name:   "netrc.machine",
			Usage:  "netrc machine",
			EnvVar: "DRONE_NETRC_MACHINE",
		},
		cli.StringFlag{
			Name:   "netrc.username",
			Usage:  "netrc username",
			EnvVar: "DRONE_NETRC_USERNAME",
		},
		cli.StringFlag{
			Name:   "netrc.password",
			Usage:  "netrc password",
			EnvVar: "DRONE_NETRC_PASSWORD",
		},
		cli.StringFlag{
			Name:   "role_arn_to_assume",
			Usage:  "A role to assume before running the terraform commands",
			EnvVar: "PLUGIN_ROLE_ARN_TO_ASSUME",
		},
		cli.StringFlag{
			Name:   "root_dir",
			Usage:  "The root directory where the terraform files live. When unset, the top level directory will be assumed",
			EnvVar: "PLUGIN_ROOT_DIR",
		},
		cli.BoolFlag{
			Name:   "sensitive",
			Usage:  "whether or not to suppress terraform commands to stdout",
			EnvVar: "PLUGIN_SENSITIVE",
		},
		cli.StringFlag{
			Name:   "tf.version",
			Usage:  "terraform version to use",
			EnvVar: "PLUGIN_TF_VERSION",
		},
		cli.StringFlag{
			Name:   "tf_data_dir",
			Usage:  "changes the location where Terraform keeps its per-working-directory data, such as the current remote backend configuration",
			EnvVar: "PLUGIN_TF_DATA_DIR",
		},
		cli.StringFlag{
			Name:   "env_prefix",
			Usage:  "the environment variable prefix",
			EnvVar: "PLUGIN_ENV_PREFIX",
			Value:  "TF_OUTPUT_",
		},
		cli.BoolFlag{
			Name:   "export_envs",
			Usage:  "export environment variables in env file",
			EnvVar: "PLUGIN_EXPORT_ENVS",
		},
		cli.StringFlag{
			Name:   "envfile",
			Usage:  "the env filename",
			EnvVar: "PLUGIN_ENVFILE",
			Value:  ".env",
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func run(c *cli.Context) error {
	logrus.WithFields(logrus.Fields{
		"Revision": revision,
	}).Info("Drone Terraform Output to Env Plugin Version")

	initOptions := InitOptions{}
	json.Unmarshal([]byte(c.String("init_options")), &initOptions)

	plugin := Plugin{
		Config: Config{
			InitOptions:      initOptions,
			Cacert:           c.String("ca_cert"),
			Sensitive:        c.Bool("sensitive"),
			RoleARN:          c.String("role_arn_to_assume"),
			RootDir:          c.String("root_dir"),
			TerraformDataDir: c.String("tf_data_dir"),
			EnvFile:          c.String("envfile"),
			ExportEnvs:       c.Bool("export_envs"),
			EnvPrefix:        c.String("env_prefix"),
		},
		Netrc: Netrc{
			Login:    c.String("netrc.username"),
			Machine:  c.String("netrc.machine"),
			Password: c.String("netrc.password"),
		},
		Terraform: Terraform{
			Version: c.String("tf.version"),
		},
	}

	return plugin.Exec()
}
