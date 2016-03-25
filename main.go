package main

import (
    "net/http/cookiejar"
    "net/http"
    "io/ioutil"
    "fmt"
    "io"
    "net/url"
    "strings"
    "regexp"
    "errors"
    "github.com/larspensjo/config"
    "flag"
    "time"
)

const (
    MHHOST = "http://www.11mh.net"
    REX_FORMHASH = "<input type=\"hidden\" name=\"formhash\" value=\"(.*?)\" />"
    REX_SIGN = "<div class=\"c\">\r\n(.*?) </div>"
)

var (
    configFile = flag.String("configfile", "config.ini", "General configuration file")
)

type User11 struct {
    userName string
    password string
    Client   *http.Client
}

func (this *User11)send(apiUri string, body io.Reader) (response []byte, err error) {
    method := "GET"
    if body != nil {
        method = "POST"
    }
    req, err := http.NewRequest(method, apiUri, body)
    if err != nil {
        return
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    resp, err := this.Client.Do(req)
    if err != nil {
        return
    }
    defer resp.Body.Close()
    response, err = ioutil.ReadAll(resp.Body)
    if err != nil {

    }
    return
}

func (this *User11)doLogin() (err error) {
    loginUrl := MHHOST + "/member.php?mod=logging&action=login&loginsubmit=yes&infloat=yes&lssubmit=yes&inajax=1"
    data := url.Values{
        "username": []string{this.userName},
        "password": []string{this.password},
        "quickforward": []string{"yes"},
        "handlekey": []string{"ls"},
    }
    resp, err := this.send(loginUrl, strings.NewReader(data.Encode()))
    if err != nil {
        return
    }
    ret := string(resp)
    match, err := regexp.MatchString("window.location.href='(.*?)';", ret)
    if !match {
        err = errors.New("账号密码错误")
    }
    return
}

func (this *User11)getFormHash() (formHash string, err error) {
    resp, err := this.send(MHHOST, nil)
    ret := string(resp)

    if err != nil {
        return

    }
    r, err := regexp.Compile(REX_FORMHASH)
    formHash = r.FindStringSubmatch(ret)[1]
    return
}

func (this *User11)doSign(todaySay string) (signInfo string, err error) {
    formHash, err := this.getFormHash()
    if err != nil {
        return
    }
    signUrl := MHHOST + "/plugin.php?id=dsu_paulsign:sign&operation=qiandao&formhash=" + formHash + "&qdmode=1&fastreply=0&qdxq=kx&infloat=yes&handlekey=dsu_paulsign&inajax=1&ajaxtarget=fwin_content_dsu_paulsign"
    data := url.Values{
        "todaysay": []string{todaySay},
    }
    resp, err := this.send(signUrl, strings.NewReader(data.Encode()))
    if err != nil {
        return
    }
    signInfo, err = this.filterSignInfo(string(resp))
    return
}

func (this *User11)filterSignInfo(txt string) (signInfo string, err error) {
    r, err := regexp.Compile(REX_SIGN)
    signInfo = r.FindStringSubmatch(txt)[1]
    return
}
func (this *User11)loginAndSign(todaysay string) (info string, err error) {
    err = this.doLogin()
    if err != nil {
        return
    }
    info, err = this.doSign(todaysay)
    if err != nil {
        return
    }
    return
}

func NewUser11(userName, password string) (user11 *User11, err error) {
    jar, err := cookiejar.New(nil)
    if err != nil {
        return
    }

    user11 = &User11{
        userName: userName,
        password: password,
        Client: &http.Client{
            Jar:jar,
        },
    }
    return
}

func main() {
    flag.Parse()
    cfg, err := config.ReadDefault(*configFile)
    if err != nil {
        fmt.Println("Fail to find", *configFile, err)
    }
    var userInfo = make(map[string]string)
    if cfg.HasSection("11mh") {
        section, err := cfg.SectionOptions("11mh")
        if err == nil {
            for _, v := range section {
                options, err := cfg.String("11mh", v)
                if err == nil {
                    userInfo[v] = options
                }
            }
        }
    }
    user, err := NewUser11(userInfo["username"], userInfo["password"])
    if err != nil {
        fmt.Println("实例化错误", err)
        return
    }
    info, err := user.loginAndSign(userInfo["say"])
    if err != nil {
        fmt.Println("签到错误", err)
        return
    }
    go func() {
        fmt.Println(info)
    }()
    timer := time.NewTimer(3 * time.Second)
    <-timer.C
}
