# GoTOTP
Go Google OTP Generator UI

A simple application to generate OTP code for a Google two-step login, based on a secret key.

It can be build and run on Linux, Mac and Windows.

To build the application:

go get github.com/andlabs/ui

go build src/GoTOTP.go (for Windows, add -ldflags "-H windowsgui" to hide console)

