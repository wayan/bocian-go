package bocian

import (
	"errors"
	"fmt"
	"github.com/urfave/cli/v2"
	"github.com/wayan/mergeexp"
	"log"
	"os"
	"regexp"
)

/*
   data for OCP, Cow, BModel
*/
type bocianapp struct {
	target               string
	bitbucketrepo        string
	defaultDeploymentKey string
	hasTest2             bool
	defaultDir           string
	deployUrl            map[string]string
	deployBranch         map[string]string
}

var OCP = &bocianapp{
	target:               "OCP",
	bitbucketrepo:        "gudang/gts-ocp",
	defaultDeploymentKey: "absbot_rsa",
	hasTest2:             true,
	defaultDir:           "ocp-demo",
	deployUrl: map[string]string{
		"TEST1": "ocplus@rztvnode404.cz.tmo:~/deploy/git/OCP.git",
		"TEST2": "ocplus@rztvnode435.cz.tmo:~/deploy/git/OCP.git",
		"PROD":  "ocplus@app-ocp.cz.tmo:~/deploy/git/OCP.git",
	},
	deployBranch: map[string]string{
		"TEST1": "demo",
		"TEST2": "ng",
		"PROD":  "PRODUCTION",
	},
}

var Cow = &bocianapp{
	target:               "Cow",
	bitbucketrepo:        "gudang/gts-cow",
	defaultDeploymentKey: "absbot_dsa",
	hasTest2:             false,
	defaultDir:           "cow-demo",
	deployUrl: map[string]string{
		"TEST1": "cowboy@rztvnode404.cz.tmo:~/deploy/git/OCP.git",
		"PROD":  "cowboy@app-ocp.cz.tmo:~/deploy/git/OCP.git",
	},
	deployBranch: map[string]string{
		"TEST1": "demo",
		"PROD":  "PRODUCTION",
	},
}

var BModel = &bocianapp{
	target:               "BModel",
	bitbucketrepo:        "gudang/gts-bmodel",
	defaultDeploymentKey: "absbot_dsa",
	hasTest2:             false,
	defaultDir:           "bmodel-demo",
	deployUrl: map[string]string{
		"TEST1": "bmodel@rztvnode404.cz.tmo:~/deploy/git/OCP.git",
		"TEST2": "bmodel@rztvnode435.cz.tmo:~/deploy/git/OCP.git",
		"PROD":  "bmodel@app-ocp.cz.tmo:~/deploy/git/OCP.git",
	},
	deployBranch: map[string]string{
		"TEST1": "demo",
		"TEST2": "demo",
		"PROD":  "PRODUCTION",
	},
}

type NilLogger struct{}

func (ba bocianapp) Info(s string) {
	fmt.Println(ba.target + " merge experimental: " + s)
}

// runs both build and deploy
func (ba bocianapp) run(c *cli.Context, run_build bool, run_deploy bool) error {
	me := &mergeexp.MergeExp{Logger: ba}
	err := ba.prepareDir(c, me)
	if err == nil && run_build {
		err = ba.prepareBitBucket(c, me)
	}

	if err != nil {
		return err
	}
	me = me.Init()

	// initializes git repo
	err = me.GitInit()
	if err != nil {
		return err
	}

	ba.Info(fmt.Sprintf("start in '%s'", me.Dir))

	tag := "experimental"
	env := "TEST1"
	localBranch := "experimental"
	if ba.hasTest2 && c.Bool("test2") {
		env = "TEST2"
		localBranch = localBranch + "-" + env
	}

	remoteBranch, err := me.BitBucketGit().FetchBranch(ba.bitbucketrepo, localBranch)
	if err != nil {
		return err
	}

	if run_build {
		startBranch, err := me.BitBucketGit().FetchBranch(ba.bitbucketrepo, "develop")
		if err != nil {
			return err
		}

		ba.Info(fmt.Sprintf("start branch '%s' from '%s'", localBranch, startBranch.Name))

		err = ba.startbranch(c, me, localBranch, startBranch)
		if err != nil {
			return err
		}

		ba.Info(fmt.Sprintf("fetch pull request branches"))

		// fetches branches to deploy
		branches, err := me.BitBucketGit().FetchPRBranches(
			ba.bitbucketrepo,
			[]string{"master", "develop"},
			[]string{tag, env + "-" + tag},
		)
		if err != nil {
			return err
		}

		ba.Info(fmt.Sprintf("merge pull request branches"))
		err = me.MergeBranches(branches)
		if err != nil {
			return err
		}

		/* final commit */
		ba.Info(fmt.Sprintf("create final commit"))
		err = me.FinalCommit(remoteBranch)
		if err != nil {
			return err
		}

		/* push to experimental */
		ba.Info(fmt.Sprintf("push to %s %s", remoteBranch.Remote, remoteBranch.Localname))
		err = me.Command("git", "push", "-f", remoteBranch.Remote, localBranch+":"+remoteBranch.Localname).Run()
		if err != nil {
			return err
		}
	}

	// if run_build is true the current branch is ready - no need to fetch it again
	if run_deploy {
		if !run_build {
			err = ba.startbranch(c, me, localBranch, remoteBranch)
			if err != nil {
				return err
			}
		}
		deployRemote, err := me.GitRemotes().CreateRemote(ba.deployUrl[env], env)
		if err != nil {
			return err
		}

		ba.Info(fmt.Sprintf("deploy to %s %s", deployRemote, ba.deployBranch[env]))
		err = me.Command("git", "push", "-f", deployRemote, localBranch+":"+ba.deployBranch[env]).Run()
		if err != nil {
			return err
		}
	}

	return err
}

func (ba bocianapp) startbranch(c *cli.Context, me *mergeexp.MergeExp, localBranch string, startBranch *mergeexp.Branch) error {
	var f func(tryreset bool) error
	f = func(tryreset bool) error {
		// starting the branch
		err := me.Command("git", "checkout", "-B", localBranch, startBranch.Name).Run()

		// if the index is broken and I work on experimental branch, I can easily call reset
		if err != nil && tryreset {
			currentbranch, err := mergeexp.OutputLine(me.Command("git", "branch", "--show-current"))
			if err != nil {
				return err
			}

			if regexp.MustCompile("^experimental").MatchString(currentbranch) {
				// clearing
				err = me.Command("git", "reset", "--hard").Run()
				if err != nil {
					return err
				}

				/* trying again */
				return f(false)
			}
		}

		return err
	}

	return f(true)
}

/*
   returns full mergeexp.MergeExp struct
*/
func (ba bocianapp) prepareDir(c *cli.Context, me *mergeexp.MergeExp) error {

	// directory
	dir := c.String("dir")

	if dir != "" {
		// if supplied directory must exists
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			return errors.New(fmt.Sprintf("Directory '%s' does not exist or it is a file", dir))
		}
	} else {
		// default directory can be created
		homedir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		dir = homedir + "/.bocian/" + ba.defaultDir
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			log.Fatal(fmt.Sprintf("Problem creating directory '%s'", err))
		}
	}
	me.Dir = dir
	return nil
}

func (ba bocianapp) prepareBitBucket(c *cli.Context, me *mergeexp.MergeExp) error {
	// deployment key
	deployment_key := c.String("deployment_key")
	if deployment_key != "" {
		/* file must exist */
		_, err := os.Stat(deployment_key)
		if err != nil {
			return err
		}
	} else {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		deployment_key = homedir + "/.ssh/" + ba.defaultDeploymentKey
		_, err = os.Stat(deployment_key)
		if os.IsNotExist(err) {
			deployment_key = ""
		} else if err != nil {
			return err
		}
	}
	me.BitBucketDeploymentKey = deployment_key

	/* BitBucketUsername, BitBucketPassword */
	user := c.String("user")
	if user == "" {
		user = os.Getenv("BOCIAN_USER")
	}
	if user == "" {
		return errors.New("No user supplied")
	}

	password := c.String("password")
	if password == "" {
		password = os.Getenv("BOCIAN_PASSWORD")
	}
	if password == "" {
		return errors.New("No password supplied")
	}
	me.BitBucketUsername = user
	me.BitBucketPassword = password

	return nil
}
