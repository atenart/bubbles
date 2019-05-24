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

package main

import (
	"flag"
	"log"

	"github.com/atenart/bubbles/db"
	"github.com/atenart/bubbles/httpserver"
	"github.com/atenart/bubbles/i18n"
	"github.com/atenart/bubbles/sendmail"
)

var (
	bind       = flag.String("bind", ":8000", "Address and port to bind to.")
	data       = flag.String("data", "data/", "Path to the data (will contain the db file as well).")
	noSignUp   = flag.Bool("no-signup", false, "Disable registration of new users.")
	smtpServer = flag.String("smtp-server", "localhost:25", "SMTP server address and port.")
	sender     = flag.String("email-from", "no-reply@bubbles", "Sender e-mail to use.")
	// Development options
	debug      = flag.Bool("debug", false, "Launch in debug mode.")
	skipLogin  = flag.Bool("skip-login", false, "Skip login and force uid to 1.")
)

func main() {
	flag.Parse()

	// FIXME: salt.
	db, err := db.Open(*data, []byte{0xc9, 0x16, 0x50, 0xff, 0x01, 0x8c, 0xe1, 0x0a})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sendmail := sendmail.Init(*smtpServer, *sender)

	i18n, err := i18n.Init()
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(httpserver.Serve(*bind, db, sendmail, i18n,
				   *noSignUp, *debug, *skipLogin))
}
