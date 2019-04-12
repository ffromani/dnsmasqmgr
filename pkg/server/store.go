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

// the package dhcphosts provides utilities to work with configuration in the dhcphosts
// (see man 8 dnsmasq) format
package server

import (
	"io/ioutil"
	"log"
)

func (dmm *DNSMasqMgr) requestStore() {
	dmm.flushChan <- true
}

func (dmm *DNSMasqMgr) storeLoop() {
	for {
		flush := <-dmm.flushChan
		if !flush {
			return
		}

		err := dmm.Store()
		if err != nil {
			log.Printf("store failed: %v", err)
		}
	}

	dmm.doneChan <- true
}

func (dmm *DNSMasqMgr) Store() error {
	var err error

	dmm.lock.Lock()
	defer dmm.lock.Unlock()

	err = ioutil.WriteFile(dmm.hostsPath, []byte(dmm.nameMap.String()), dmm.hostsInfo.Mode())
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(dmm.leasesPath, []byte(dmm.addrMap.String()), dmm.leasesInfo.Mode())
	if err != nil {
		return err
	}

	return nil
}
