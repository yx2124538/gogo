package fingers

import (
	"encoding/json"
	"fmt"
	"github.com/chainreactors/gogo/v2/pkg/dsl"
	"github.com/chainreactors/logs"
	"regexp"
	"strings"
)

// common struct
func decode(s string) []byte {
	var bs []byte
	if s[:4] == "b64|" {
		bs = dsl.Base64Decode(s[4:])
	} else if s[:5] == "hex|" {
		bs = dsl.UnHexlify(s[5:])
	} else {
		bs = []byte(s)
	}
	return bs
}

func compileRegexp(s string) (*regexp.Regexp, error) {
	reg, err := regexp.Compile(s)
	if err != nil {
		return nil, err
	}
	return reg, nil
}

func mapToString(m map[string]interface{}) string {
	if m == nil || len(m) == 0 {
		return ""
	}
	var s string
	for k, v := range m {
		s += fmt.Sprintf(" %s:%s ", k, v.(string))
	}
	return s
}

type Finger struct {
	Name        string   `yaml:"name" json:"name"`
	Protocol    string   `yaml:"protocol,omitempty" json:"protocol"`
	DefaultPort []string `yaml:"default_port,omitempty" json:"default_port,omitempty"`
	Focus       bool     `yaml:"focus,omitempty" json:"focus,omitempty"`
	Rules       Rules    `yaml:"rule,omitempty" json:"rule,omitempty"`
}

func (finger *Finger) Compile(portHandler func([]string) []string) error {
	if finger.Protocol == "" {
		finger.Protocol = "http"
	}

	if len(finger.DefaultPort) == 0 {
		if finger.Protocol == "http" {
			finger.DefaultPort = []string{"80"}
		}
	} else {
		finger.DefaultPort = portHandler(finger.DefaultPort)
	}

	err := finger.Rules.Compile()
	if err != nil {
		return err
	}
	return nil
}

func (finger *Finger) ToResult(hasFrame, hasVuln bool, res string, index int) (frame *Framework, vuln *Vuln) {
	if index+1 > len(finger.Rules) {
		return nil, nil
	}

	if hasFrame {
		if res != "" {
			frame = &Framework{Name: finger.Name, Version: res}
		} else if finger.Rules[index].Version != "" {
			frame = &Framework{Name: finger.Name, Version: res}
		} else {
			frame = &Framework{Name: finger.Name}
		}
	}

	if hasVuln {
		if finger.Rules[index].Vuln != "" {
			vuln = &Vuln{Name: finger.Rules[index].Vuln, Severity: "high"}
		} else if finger.Rules[index].Info != "" {
			vuln = &Vuln{Name: finger.Rules[index].Info, Severity: "info"}
		} else {
			vuln = &Vuln{Name: finger.Name, Severity: "info"}
		}
	}
	return frame, vuln
}

func (finger *Finger) Match(content string, level int, sender func([]byte) (string, bool)) (*Framework, *Vuln, bool) {
	// 只进行被动的指纹判断, 将无视rules中的senddata字段
	for i, rule := range finger.Rules {
		var ishttp bool
		var isactive bool
		if finger.Protocol == "http" {
			ishttp = true
		}
		var c string
		var ok bool
		if level >= rule.Level && rule.SendData != nil {
			logs.Log.Debugf("active match with %s", rule.SendDataStr)
			c, ok = sender(rule.SendData)
			if ok {
				isactive = true
				content = strings.ToLower(c)
			}
		}
		hasFrame, hasVuln, res := RuleMatcher(rule, content, ishttp)
		if hasFrame {
			frame, vuln := finger.ToResult(hasFrame, hasVuln, res, i)
			if finger.Focus {
				frame.IsFocus = true
			}
			if isactive && hasFrame && ishttp {
				frame.Data = c
			}
			if frame.Version == "" && rule.Regexps.CompiledVersionRegexp != nil {
				for _, reg := range rule.Regexps.CompiledVersionRegexp {
					res, _ := compiledMatch(reg, content)
					if res != "" {
						frame.Version = res
						break
					}
				}
			}
			if isactive {
				frame.From = "active"
			}
			return frame, vuln, true
		}
	}
	return nil, nil, false
}

type Regexps struct {
	Body                  []string         `yaml:"body,omitempty" json:"body,omitempty"`
	MD5                   []string         `yaml:"md5,omitempty" json:"md5,omitempty"`
	MMH3                  []string         `yaml:"mmh3,omitempty" json:"mmh3,omitempty"`
	Regexp                []string         `yaml:"regexp,omitempty" json:"regexp,omitempty"`
	Version               []string         `yaml:"version,omitempty" json:"version,omitempty"`
	CompliedRegexp        []*regexp.Regexp `yaml:"-" json:"-"`
	CompiledVulnRegexp    []*regexp.Regexp `yaml:"-" json:"-"`
	CompiledVersionRegexp []*regexp.Regexp `yaml:"-" json:"-"`
	Header                []string         `yaml:"header,omitempty" json:"header,omitempty"`
	Vuln                  []string         `yaml:"vuln,omitempty" json:"vuln,omitempty"`
}

func (r *Regexps) RegexpCompile() error {
	for _, reg := range r.Regexp {
		creg, err := compileRegexp("(?i)" + reg)
		if err != nil {
			return err
		}
		r.CompliedRegexp = append(r.CompliedRegexp, creg)
	}

	for _, reg := range r.Vuln {
		creg, err := compileRegexp("(?i)" + reg)
		if err != nil {
			return err
		}
		r.CompiledVulnRegexp = append(r.CompiledVulnRegexp, creg)
	}

	for _, reg := range r.Version {
		creg, err := compileRegexp(reg)
		if err != nil {
			return err
		}
		r.CompiledVersionRegexp = append(r.CompiledVersionRegexp, creg)
	}

	for i, b := range r.Body {
		r.Body[i] = strings.ToLower(b)
	}

	for i, h := range r.Header {
		r.Header[i] = strings.ToLower(h)
	}
	return nil
}

type Favicons struct {
	Mmh3 []string `yaml:"mmh3,omitempty" json:"mmh3,omitempty"`
	Md5  []string `yaml:"md5,omitempty" json:"md5,omitempty"`
}

type Rule struct {
	Version     string    `yaml:"version,omitempty" json:"version,omitempty"`
	Favicon     *Favicons `yaml:"favicon,omitempty" json:"favicon,omitempty"`
	Regexps     *Regexps  `yaml:"regexps,omitempty" json:"regexps,omitempty"`
	SendDataStr string    `yaml:"send_data,omitempty" json:"send_data,omitempty"`
	SendData    senddata  `yaml:"-,omitempty" json:"-,omitempty"`
	Info        string    `yaml:"info,omitempty" json:"info,omitempty"`
	Vuln        string    `yaml:"vuln,omitempty" json:"vuln,omitempty"`
	Level       int       `yaml:"level,omitempty" json:"level,omitempty"`
}

func (rule *Rule) Match(content string, ishttp bool) (bool, bool, string) {
	// 漏洞匹配优先
	if rule.Regexps == nil {
		return false, false, ""
	}
	for _, reg := range rule.Regexps.CompiledVulnRegexp {
		res, ok := compiledMatch(reg, content)
		if ok {
			return true, true, res
		}
	}

	var body, header string
	if ishttp {
		cs := strings.Index(content, "\r\n\r\n")
		if cs != -1 {
			body = content[cs+4:]
			header = content[:cs]
		}
	} else {
		body = content
	}

	// body匹配
	for _, bodyReg := range rule.Regexps.Body {
		if strings.Contains(body, bodyReg) {
			return true, false, ""
		}
	}

	// 正则匹配
	for _, reg := range rule.Regexps.CompliedRegexp {
		res, ok := compiledMatch(reg, content)
		if ok {
			return true, false, res
		}
	}

	// MD5 匹配
	for _, md5s := range rule.Regexps.MD5 {
		if md5s == dsl.Md5Hash([]byte(content)) {
			return true, false, ""
		}
	}

	// mmh3 匹配
	for _, mmh3s := range rule.Regexps.MMH3 {
		if mmh3s == dsl.Mmh3Hash32([]byte(content)) {
			return true, false, ""
		}
	}

	// http头匹配, http协议特有的匹配
	if !ishttp {
		return false, false, ""
	}

	for _, headerReg := range rule.Regexps.Header {
		if strings.Contains(header, headerReg) {
			return true, false, ""
		}
	}
	return false, false, ""
}

type Rules []*Rule

func (rs Rules) Compile() error {
	for _, r := range rs {
		if r.SendDataStr != "" {
			r.SendData = decode(r.SendDataStr)
			if r.Level == 0 {
				r.Level = 1
			}
		}

		if r.Regexps != nil {
			err := r.Regexps.RegexpCompile()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type senddata []byte

func (d senddata) IsNull() bool {
	if len(d) == 0 {
		return true
	}
	return false
}

type FingerMapper map[string][]*Finger

func (fm FingerMapper) GetFingers(port string) []*Finger {
	return fm[port]
}

type Fingers []*Finger

func (fs Fingers) Contain(f *Finger) bool {
	for _, finger := range fs {
		if f == finger {
			return true
		}
	}
	return false
}

func (fs Fingers) GroupByPort() FingerMapper {
	fingermap := make(FingerMapper)
	for _, f := range fs {
		for _, port := range f.DefaultPort {
			fingermap[port] = append(fingermap[port], f)
		}
	}
	return fingermap
}

func LoadFingers(content []byte) (fingers Fingers, err error) {
	err = json.Unmarshal(content, &fingers)
	if err != nil {
		return nil, err
	}
	return fingers, nil
}