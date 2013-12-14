package github

import (
	"code.google.com/p/go.crypto/ssh/terminal"
	"encoding/json"
	"fmt"
	"github.com/howeyc/gopass"
	"github.com/jingweno/gh/utils"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

var (
	defaultConfigsFile = filepath.Join(os.Getenv("HOME"), ".config", "gh")
)

type Credentials struct {
	Host        string `json:"host"`
	User        string `json:"user"`
	AccessToken string `json:"access_token"`
}

type Configs struct {
	Credentials []Credentials
}

func (c *Configs) PromptFor(host string) *Credentials {
	cc := c.Find(host)
	if cc == nil {
		user := c.PromptForUser()
		pass := c.PromptForPassword(host, user)

		client := &GitHub{Project: &Project{Host: host}}
		token, err := client.FindOrCreateToken(user, pass, "")
		if err != nil {
			re := regexp.MustCompile("two-factor authentication OTP code")
			if re.MatchString(fmt.Sprintf("%s", err)) {
				code := c.PromptForOTP()
				token, err = client.FindOrCreateToken(user, pass, code)
			}
		}
		utils.Check(err)

		cc = &Credentials{Host: host, User: user, AccessToken: token}
		c.Credentials = append(c.Credentials, *cc)
		err = saveTo(configsFile(), c.Credentials)
		utils.Check(err)
	}

	return cc
}

func (c *Configs) PromptForUser() string {
	var user string
	fmt.Printf("%s username: ", GitHubHost)
	fmt.Scanln(&user)

	return user
}

func (c *Configs) PromptForPassword(host, user string) (pass string) {
	fmt.Printf("%s password for %s (never stored): ", host, user)
	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		pass = string(gopass.GetPasswd())
	} else {
		fmt.Scanln(&pass)
	}

	return
}

func (c *Configs) PromptForOTP() string {
	var code string
	fmt.Print("two-factor authentication code: ")
	fmt.Scanln(&code)

	return code
}

func (c *Configs) Find(host string) *Credentials {
	for _, t := range c.Credentials {
		if t.Host == host {
			return &t
		}
	}

	return nil
}

func saveTo(filename string, v interface{}) error {
	err := os.MkdirAll(filepath.Dir(filename), 0771)
	if err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	return enc.Encode(v)
}

func loadFrom(filename string, v interface{}) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	for {
		if err := dec.Decode(v); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
	}

	return nil
}

func configsFile() string {
	configsFile := os.Getenv("GH_CONFIG")
	if configsFile == "" {
		configsFile = defaultConfigsFile
	}

	return configsFile
}

func CurrentConfigs() *Configs {
	var c []Credentials
	err := loadFrom(configsFile(), &c)
	if err != nil {
		c = make([]Credentials, 0)
	}

	return &Configs{c}
}

func (c *Configs) DefaultCredentials() (credentials *Credentials) {
	if GitHubHostEnv != "" {
		credentials = c.PromptFor(GitHubHostEnv)
	} else if len(c.Credentials) == 1 {
		credentials = &c.Credentials[0]
	} else {
		credentials = c.PromptFor(DefaultHost())
	}

	return
}

// Public for testing purpose
func CreateTestConfigs(user, token string) []Credentials {
	f, _ := ioutil.TempFile("", "test-config")
	defaultConfigsFile = f.Name()

	c := []Credentials{
		{User: "jingweno", AccessToken: "123", Host: GitHubHost},
	}
	saveTo(f.Name(), &c)

	return c
}
