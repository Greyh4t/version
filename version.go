package version

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"regexp"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

var datePattern = regexp.MustCompile(`\A20[12]\d(0[1-9]|1[012])(0[1-9]|[12]\d|3[01])(\d{2}){0,3}\z`)

var keyword = map[string]string{
	"dev":      "dev",
	"develop":  "dev",
	"snapshot": "snapshot",
	"a":        "alpha",
	"alpha":    "alpha",
	"b":        "beta",
	"beta":     "beta",
	"stable":   "stable",
	"final":    "final",
	"fixed":    "fixed",
	"m":        "milestone",
	"c":        "rc",
	"rc":       "rc",
	"ga":       "ga",
	"r":        "release",
	"release":  "release",
}

var weights = map[string]string{
	"dev":       "00",
	"snapshot":  "01",
	"alpha":     "02",
	"beta":      "03",
	"stable":    "04",
	"final":     "05",
	"fixed":     "06",
	"milestone": "07",
	"rc":        "08",
	"ga":        "09",
	"release":   "10",
}

type Pre struct {
	Letter string
	Number int
}

type Version struct {
	Version string
	release []int
	date    string
	prelist []Pre
}

func (v *Version) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "version" //设置标签为小写version

	err := e.EncodeToken(start)
	if err != nil {
		return err
	}

	e.EncodeToken(xml.CharData(v.Version))
	return e.EncodeToken(start.End())
}

func (v *Version) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var chardata xml.CharData
	err := d.DecodeElement(&chardata, &start)
	if err != nil {
		return err
	}
	parseVersion(string(chardata), v)
	return nil
}

func (v *Version) UnmarshalBSONValue(t bsontype.Type, raw []byte) error {
	versionStr, rem, ok := bsoncore.ReadString(raw)
	if !ok {
		return bsoncore.NewInsufficientBytesError(raw, rem)
	}

	parseVersion(versionStr, v)
	return nil
}

func (v Version) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, v.Version), nil
}

func (v *Version) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Version)
}

func (v *Version) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, &v.Version)
	if err != nil {
		return err
	}

	parseVersion(v.Version, v)
	return nil
}

func (v *Version) Lt(version *Version) bool {
	if v.Version == "" || version.Version == "" {
		return false
	}

	return v.Compare(version) == -1
}

func (v *Version) Lte(version *Version) bool {
	if v.Version == "" || version.Version == "" {
		return false
	}

	r := v.Compare(version)
	return r == 0 || r == -1
}

func (v *Version) Gt(version *Version) bool {
	if v.Version == "" || version.Version == "" {
		return false
	}

	return v.Compare(version) == 1
}

func (v *Version) Gte(version *Version) bool {
	if v.Version == "" || version.Version == "" {
		return false
	}

	r := v.Compare(version)
	return r == 0 || r == 1
}

func (v *Version) Eq(version *Version) bool {
	if v.Version == "" || version.Version == "" {
		return false
	}

	return v.Compare(version) == 0
}

func (v *Version) Compare(version *Version) int {
	// release
	l := len(v.release)
	if len(version.release) < l {
		l = len(version.release)
	}

	for i := 0; i < l; i++ {
		if v.release[i] > version.release[i] {
			return 1
		} else if v.release[i] < version.release[i] {
			return -1
		}
	}

	if len(v.release) > len(version.release) {
		return 1
	} else if len(v.release) < len(version.release) {
		return -1
	}

	// date
	if v.date != "" && version.date != "" {
		if v.date > version.date {
			return 1
		} else if v.date < version.date {
			return -1
		}
	}

	// pre
	l = len(v.prelist)
	if len(version.prelist) > l {
		l = len(version.prelist)
	}

	for i := 0; i < l; i++ {
		var pre1, pre2 Pre

		if i < len(v.prelist) {
			pre1 = v.prelist[i]
		} else {
			pre1 = Pre{"release", 0}
		}

		if i < len(version.prelist) {
			pre2 = version.prelist[i]
		} else {
			pre2 = Pre{"release", 0}
		}

		r := comparePre(pre1, pre2)
		if r != 0 {
			return r
		}
	}

	return 0
}

func parseVersion(version string, v *Version) {
	v.Version = version

	list := splitVersionParts(strings.ToLower(version))
	var releaseFlag = true
	for i := 0; i < len(list); i++ {
		part := list[i]

		if part == "_" || part == "-" {
			releaseFlag = false
			continue
		}

		// 解析 date
		if datePattern.MatchString(part) {
			part = part + strings.Repeat("0", 14-len(part))
			v.date = part
			releaseFlag = false
			continue
		}

		// 解析 release
		if releaseFlag {
			n, err := strconv.Atoi(part)
			if err != nil {
				releaseFlag = false
			} else {
				v.release = append(v.release, n)
				continue
			}
		}

		// 解析 pre
		if value, ok := keyword[part]; ok {
			part = value
		}

		var n int
		if _, err := strconv.Atoi(part); err != nil {
			if i+1 < len(list) {
				if !datePattern.MatchString(list[i+1]) {
					n, err = strconv.Atoi(list[i+1])
					if err == nil {
						i++
					}
				}
			}
		}

		v.prelist = append(v.prelist, Pre{part, n})
	}

	// 删除尾部 0
	if len(v.release) > 0 {
		index := len(v.release)
		for i := index - 1; i >= 0; i-- {
			if v.release[i] == 0 {
				index = i
			} else {
				break
			}
		}
		if index < len(v.release) {
			v.release = v.release[0:index]
		}
	}
}

func Parse(version string) *Version {
	var v = &Version{}
	parseVersion(version, v)
	return v
}

func comparePre(pre1, pre2 Pre) int {
	weight1 := pre1.Letter
	if weights[pre1.Letter] != "" {
		weight1 = weights[pre1.Letter]
	}

	weight2 := pre1.Letter
	if weights[pre2.Letter] != "" {
		weight2 = weights[pre2.Letter]
	}

	if weight1 > weight2 {
		return 1
	} else if weight1 < weight2 {
		return -1
	}

	if pre1.Number > pre2.Number {
		return 1
	} else if pre1.Number < pre2.Number {
		return -1
	}

	return 0
}

func splitVersionParts(version string) []string {
	var list []string
	var buf bytes.Buffer
	var lastRune = 0

	var flushBuf = func() {
		if buf.Len() > 0 {
			list = append(list, buf.String())
			buf.Reset()
		}
	}

	for _, r := range version {
		switch true {
		case '0' <= r && r <= '9':
			if lastRune != 'N' {
				flushBuf()
			}

			buf.WriteRune(r)
			lastRune = 'N'

		case 'a' <= r && r <= 'z':
			if lastRune != 'S' {
				flushBuf()
			}

			buf.WriteRune(r)
			lastRune = 'S'

		default:
			flushBuf()
			lastRune = 0
		}
	}

	if buf.Len() > 0 {
		list = append(list, buf.String())
		buf.Reset()
	}

	return list
}
