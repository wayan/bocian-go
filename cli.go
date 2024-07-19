package bocian

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/urfave/cli/v2"
)

func (ba bocianapp) mergeexpcli() *cli.App {
	commands := []*cli.Command{
		{
			Name:  "build",
			Usage: "only builds the experimental branch (and pushes it to BitBucket",
			Action: func(c *cli.Context) error {
				return ba.run(c, true, false)
			},
		},
		{
			Name:  "deploy",
			Usage: "only deploys the experimental brach from BitBucket to proper environment",
			Action: func(c *cli.Context) error {
				return ba.run(c, false, true)
			},
		},
	}
	flags := []cli.Flag{}

	flags = append(flags,
		&cli.BoolFlag{
			Name:  "version",
			Usage: "prints the version of the installation script",
		},
	)

	flags = append(flags,
		&cli.StringFlag{
			Name:    "deployment_key",
			Aliases: []string{"dk"},
			Usage: fmt.Sprintf(`path to deployment key used to fetch the repos from BitBucket. If not supplied, file ~/.ssh/%s is used, if exists`,
				ba.defaultDeploymentKey,
			),
			/*
			   Description:
			   fmt.Println(
			       "If not supplied, file ~/.ssh/%s is used if exists",
			       ba.defaultDeploymentKey(),
			   ),
			*/
		},
	)

	flags = append(
		flags,
		&cli.StringFlag{
			Name:    "user",
			Aliases: []string{"u"},
			Usage:   "BitBucket user name for access to REST",
			EnvVars: []string{"BOCIAN_USER"},
		},
		&cli.StringFlag{
			Name:    "password",
			Aliases: []string{"p"},
			Usage:   "BitBucket application password for access to REST",
			EnvVars: []string{"BOCIAN_PASSWORD"},
		},
	)
	/*
	       $cli->add_option( <<"END_USER");
	   user|u=s - Bitbucket username (likely email). If not supplied then environment variable BOCIAN_USER is used.
	   If the variable is not set, the username will be asked for.
	   END_USER
	       $cli->add_option( <<"END_PASSWORD");
	   password|p=s - Bitbucket user password, if not supplied then environment variable BOCIAN_PASSWORD is used.
	   If the variable is not set, the username will be asked for.
	   END_PASSWORD
	*/
	/*
	       $cli->add_option(<<"END_KEY");
	   deployment_key|dk=s - path to deployment key used to fetch the repos fro BitBucket.
	   If not supplied, uses ${\ default_deployment_key($target) }, if the file exists.
	   END_KEY
	       $cli->add_option( <<"END_USER");
	   user|u=s - Bitbucket username (likely email). If not supplied then environment variable BOCIAN_USER is used.
	   If the variable is not set, the username will be asked for.
	   END_USER
	       $cli->add_option( <<"END_PASSWORD");
	   password|p=s - Bitbucket user password, if not supplied then environment variable BOCIAN_PASSWORD is used.
	   If the variable is not set, the username will be asked for.
	   END_PASSWORD
	*/
	if ba.hasTest2 {
		flags = append(flags,
			&cli.BoolFlag{Name: "test2", Usage: "deploys to TEST2 environment (default is TEST1)"},
		)
	}
	desc := fmt.Sprintf(
		`Builds the experimental version of %s and deploys it to demo environment. The process:

   * builds experimental branch and pushes it to BitBucket to %s/experimental branch

   * deploys (push) the experimental branch to test environment
`, ba.target, ba.bitbucketrepo)

	app := &cli.App{
		Name:        ba.target + "-MergeExperimental",
		Usage:       ` experimental merge and deploy!`,
		Description: desc,
		Flags:       flags,
		Action: func(c *cli.Context) error {
			if c.Bool("version") {
				info, _ := debug.ReadBuildInfo()
				for _, dep := range info.Deps {
					fmt.Printf("%+w\n", dep)
				}
				fmt.Println(ba.GetVersion())
				return nil
			}
			return ba.run(c, true, true)
		},
		Commands: commands,
	}
	return app
}

func (ba bocianapp) GetVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	path := "github.com/wayan/bocian-go"
	for _, module := range info.Deps {
		if module.Path == path {
			return module.Version
		}
	}
	return "unknown"
}

func (ba bocianapp) RunMergeExp() {
	app := ba.mergeexpcli()
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
