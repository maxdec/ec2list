# ec2list

Terminal application that displays your AWS EC2 instances and more.

![Screenshot](https://raw.githubusercontent.com/maxdec/ec2list/master/screenshot.png)

## Installation

If you have golang installed:

```bash
$ go get github.com/maxdec/ec2list
```

Otherwise you can download the [latest release](https://github.com/maxdec/ec2list/releases) (for OSX), and put it in your `$PATH` (make sure it's `chmod +x`'ed).

You need to following environment variables:

```
AWS_ACCESS_KEY_ID=XXX
AWS_SECRET_ACCESS_KEY=XXX
AWS_REGION=eu-central-1
```

## Usage

```bash
$ ec2list
```

That's it :-)
