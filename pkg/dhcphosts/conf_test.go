/*
 * Copyright 2019 Francesco Romani - fromani/gmail
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of this
 * software and associated documentation files (the "Software"), to deal in the Software
 * without restriction, including without limitation the rights to use, copy, modify,
 * merge, publish, distribute, sublicense, and/or sell copies of the Software, and to
 * permit persons to whom the Software is furnished to do so, subject to the following
 * conditions:
 *
 * The above copyright notice and this permission notice shall be included in all copies
 * or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED,
 * INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A
 * PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
 * HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
 * OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
 * SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

package dhcphosts

import (
	"io/ioutil"
	"os"
	"strings"

	"testing"
)

func TestBindingParseError(t *testing.T) {
	var err error

	_, err = ParseBindingString("")
	if err != ErrBadBindingFormat {
		t.Errorf("unexpected error: %v", err)
	}

	_, err = ParseBindingString("01:23:45:67:89:ab")
	if err != ErrBadBindingFormat {
		t.Errorf("unexpected error: %v", err)
	}

	_, err = ParseBindingString("1.1.1.1")
	if err != ErrBadBindingFormat {
		t.Errorf("unexpected error: %v", err)
	}

	// legal in dnsmasq.conf, but unsupported yet
	_, err = ParseBindingString("01:23:45:67:89:ab,1.1.1.1,extra")
	if err != ErrBadBindingFormat {
		t.Errorf("unexpected error: %v", err)
	}

	_, err = ParseBindingString("malformed_mac,1.1.1.1")
	if err != ErrBadHWAddrFormat {
		t.Errorf("unexpected error: %v", err)
	}

	_, err = ParseBindingString("01:23:45:67:89:ab,malformed_IP")
	if err != ErrBadIPFormat {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBindingParseRoundTrip(t *testing.T) {
	s := "01:23:45:67:89:ab,1.1.1.1"
	b, err := ParseBindingString(s)
	if err != nil {
		t.Errorf("parsing b1: unexpected error: %v", err)
	}
	x := b.String()
	if s != x {
		t.Errorf("failed roundtrip: %s %s", s, x)
	}
}

func TestBindingEqual(t *testing.T) {
	b1, err := ParseBinding("01:23:45:67:89:ab", "1.1.1.1")
	if err != nil {
		t.Errorf("parsing b1: unexpected error: %v", err)
	}
	b2, err := ParseBinding("01:23:45:67:89:ab", "2.2.2.2")
	if err != nil {
		t.Errorf("parsing b2: unexpected error: %v", err)
	}
	b3, err := ParseBinding("fe:dc:ba:98:76:54", "2.2.2.2")
	if err != nil {
		t.Errorf("parsinbg b3: unexpected error: %v", err)
	}
	if b1.Equal(b2) || b1.Equal(b3) {
		t.Errorf("unexpectedly equal: %v %v %v", b1, b2, b3)
	}
	if !b1.EqualHW(b2.HW) {
		t.Errorf("unexpectedly different HW: %v %v", b1, b2)
	}
	if !b2.EqualIP(b3.IP) {
		t.Errorf("unexpectedly different IP: %v %v", b1, b2)
	}
}

func TestConfEmpty(t *testing.T) {
	c := &Conf{}
	L := c.Len()
	if L != 0 {
		t.Errorf("length %v not zero", L)
	}
	var err error
	_, err = c.GetByIP("1.1.1.1")
	if err != ErrIPAddrNotFound {
		t.Errorf("unexpected error: %v", err)
	}
	_, err = c.GetByHWAddr("01:23:45:67:89:ab")
	if err != ErrHWAddrNotFound {
		t.Errorf("unexpected error: %v", err)
	}
}

const testData string = "" +
	"01:23:45:67:89:ab,1.1.1.1\n" +
	"fe:dc:ba:98:76:54,2.2.2.2\n" +
	""

func TestConfLoadFile(t *testing.T) {
	tmpfile, err := mktmpfile(testData)
	if err != nil {
		t.Errorf("unexpected error creating tmpfile: %v", err)
	}
	defer cleanup(tmpfile)

	m, err := Parse(tmpfile)
	if err != nil {
		t.Errorf("unexpected error reading tmpfile: %v", err)
	}
	L := m.Len()
	if L != 2 {
		t.Errorf("unexpected number of entries: %v", L)
	}
}

func TestFindWithContent(t *testing.T) {
	ref, err := ParseBindingString("fe:dc:ba:98:76:54,1.1.1.1")
	if err != nil {
		t.Errorf("unexpected error parsing the ref data: %v", err)
	}

	sr := strings.NewReader(testData)
	m, err := Parse(sr)
	if err != nil {
		t.Errorf("unexpected error reading tmpfile: %v", err)
	}
	L := m.Len()
	if L != 2 {
		t.Errorf("unexpected number of entries: %v", L)
	}

	b1, err := m.GetByHWAddr("01:23:45:67:89:ab")
	if err != nil {
		t.Errorf("unexpected error looking by HWAddr: %v", err)
	}
	if !b1.EqualIP(ref.IP) {
		t.Errorf("ref mismatch: [%s] [%s]", b1.IP.String(), ref.IP.String())
	}

	b2, err := m.GetByIP("2.2.2.2")
	if err != nil {
		t.Errorf("unexpected error looking by I{: %v", err)
	}
	if !b2.EqualHW(ref.HW) {
		t.Errorf("ref mismatch: [%s] [%s]", b2.IP.String(), ref.IP.String())
	}
}

func TestConfDumpFile(t *testing.T) {
	sr := strings.NewReader(testData)
	m, err := Parse(sr)
	if err != nil {
		t.Errorf("unexpected error reading tmpfile: %v", err)
	}
	L := m.Len()
	if L != 2 {
		t.Errorf("unexpected number of entries: %v", L)
	}

	tmpfile, err := mktmpfile("")
	if err != nil {
		cleanup(tmpfile)
		t.Errorf("unexpected error creating tmpfile: %v", err)
	}
	_, err = tmpfile.WriteString(m.String())
	if err != nil {
		cleanup(tmpfile)
		t.Errorf("unexpected error dumping to tmpfile: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	err = tmpfile.Close()
	if err != nil {
		t.Errorf("unexpected error closing tmpfile: %v", err)
	}

	content, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		t.Errorf("unexpected error reading back tmpfile: %v", err)
	}
	if string(content) != testData {
		t.Errorf("inconsistent content: %v\n%v\n", string(content), testData)
	}
}

func cleanup(tmpfile *os.File) {
	tmpfile.Close()
	os.Remove(tmpfile.Name())
}

func mktmpfile(content string) (*os.File, error) {
	tmpfile, err := ioutil.TempFile("", "dhcphosts")
	if err != nil {
		return nil, err
	}
	if _, err := tmpfile.WriteString(content); err != nil {
		tmpfile.Close()
		return nil, err
	}
	if err := tmpfile.Sync(); err != nil {
		tmpfile.Close()
		return nil, err
	}
	if _, err := tmpfile.Seek(0, 0); err != nil {
		tmpfile.Close()
		return nil, err
	}
	return tmpfile, nil
}
