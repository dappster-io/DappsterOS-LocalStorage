package dropbox

import (
	"github.com/dappster-io/DappsterOS-LocalStorage/internal/driver"
	"github.com/dappster-io/DappsterOS-LocalStorage/internal/op"
)

const ICONURL = "./img/driver/Dropbox.svg"
const APPKEY = "tciqajyazzdygt9"
const APPSECRET = "e7gtmv441cwdf0n"

type Addition struct {
	driver.RootID
	RefreshToken   string `json:"refresh_token" required:"true" omit:"true"`
	AppKey         string `json:"app_key" type:"string" default:"tciqajyazzdygt9" omit:"true"`
	AppSecret      string `json:"app_secret" type:"string" default:"e7gtmv441cwdf0n" omit:"true"`
	OrderDirection string `json:"order_direction" type:"select" options:"asc,desc" omit:"true"`
	AuthUrl        string `json:"auth_url" type:"string" default:"https://www.dropbox.com/oauth2/authorize?client_id=tciqajyazzdygt9&redirect_uri=https://cloudoauth.files.dappsteros.app&response_type=code&token_access_type=offline&state=${HOST}%2Fv1%2Frecover%2FDropbox&&force_reapprove=true&force_reauthentication=true"`
	Icon           string `json:"icon" type:"string" default:"./img/driver/Dropbox.svg"`
	Code           string `json:"code" type:"string" help:"code from auth_url" omit:"true"`
}

var config = driver.Config{
	Name:        "Dropbox",
	OnlyProxy:   true,
	DefaultRoot: "root",
}

func init() {
	op.RegisterDriver(func() driver.Driver {
		return &Dropbox{}
	})
}
