# rm-zoterosync

I made this because I wanted to be able to get my zotero library onto my reMarkable and I did not want to use a cloud server to do so.
This is possible because you can ssh into a reMarkable and run any executable that is compiled for Linux ARMv7.

## How to run it on your reMarkable

Download the executable from [here](https://github.com/Maaarcocr/rm-zoterosync/releases/download/0.1/rm-zoterosync). Copy it on your reMarkable using `scp` and run it with `rm-zoterosync &` so that it executes in the backgroud.  

You have to specify 2 environment variables before running it on your reMarkable: `ZOTERO_USERID` (which is not your username and you can find it at [here](https://www.zotero.org/settings/keys)) and `ZOTERO_APIKEY`.

The go code will only sync your zotero collections for which a folder in your remarkable exists with the exact same name. 

## Compile from source

If you want to compile the source code on your own you have to use the Go compiler and set these environment variables before compiling: `GOOS="linux"`, `GOARCH="arm"` and `GOARM=7`
