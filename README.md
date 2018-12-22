# rm-zoterosync

I made this because I wanted to be able to get my zotero library onto my reMarkable and I did not want to use a cloud server to do so.
This is possible because you can ssh into a reMarkable and run any executable that is compiled for Linux ARMv7.

## How to run it on your reMarkable

Compile the `main.go` using `GOOS=linux GOARCH=arm GOARM=7` and `scp` it onto your reMarkable. Run it using `rm-zoterosync &` so that it run
in the backgroud. 

You have to specify 2 environment variables: `ZOTERO_USERID` (which is not your username and you can find it at [here](https://www.zotero.org/settings/keys)) and `ZOTERO_APIKEY`.

The go code will only sync your zotero collections for which a folder in your remarkable exists with the exact same name. 
