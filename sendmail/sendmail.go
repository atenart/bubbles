// Copyright (C) 2019 Antoine Tenart <antoine.tenart@ack.tf>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package sendmail

import (
	"crypto/tls"
	"net/smtp"
	"strings"
)

type Sendmail struct {
	Remote string
	Sender string
}

// Start a new Sendmail instance.
func Init(remote, sender string) *Sendmail {
	return &Sendmail{
		remote,
		sender,
	}
}

// Send a mail to an user.
func (m *Sendmail) Send(recipient, subject, body string) error {
	serverName := strings.Split(m.Remote, ":")[0]

	// Start a new TLS connection.
	tlsconfig := &tls.Config {
		InsecureSkipVerify: true,
		ServerName: serverName,
	}
	conn, err := tls.Dial("tcp", m.Remote, tlsconfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Start a connection to the remote SMTP server.
	c, err := smtp.NewClient(conn, serverName)
	if err != nil {
		return err
	}
	defer c.Close()

	// Set the sender and recipient.
	c.Mail(m.Sender)
	c.Rcpt(recipient)

	// Get an io.WriteCloser for the body.
	w, err := c.Data()
	if err != nil {
		return err
	}
	defer w.Close()

	// Write the body.
	if _, err := w.Write([]byte(body)); err != nil {
		return err
	}

	// Send the QUIT cmd.
	c.Quit()

	return nil
}
