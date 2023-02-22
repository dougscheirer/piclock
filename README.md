# piclock

A golang project for turning an rpi, a push button, and a 7-seg display into an alarm clock based off of gcal events.

### build
	go build [--tags='noaudio nobuttons']

### required packages
	portaudio19-dev
	libmpg123-dev
  python-smbus
  i2c-tools

### running as a service
	not working great yet, directories are tricky, simulations are hard, etc.

### pi setup
  * /boot/ssh to enable ssh default
  * reset password
  * set TZ/keyboard/I2C/GPU to 16M (opt)
  * set hostname
  * ssh-keygen, copy to AU
  * add local pubkey to authkeys
  (TODO: break these into git/vim and post-git clone) 
    install: vim git portaudio19-dev libmpg123-dev golang
  * git clone
    - (if using 256M board)
        - add USB as sda
        - sudo mkswap /dev/sda
        - sudo swapon /dev/sda
    - (for git shallow clone) git clone github.com/schollz/git && go build && PATH=$GOPATH/bin:$PATH 
  * go get ./... + go build
  * (TODO: make a web page for configuring the gcal api) configure the gapi token file in ~/.credentials in the web page
    form: run configure from web, redirect back to (here), save OAUTH token
  * (TODO: service install script)
  * (TODO: scripted test setup)
  
### TODO
  add simulation program for testing?
  
### tests
  * alarm:
    - load empty alarm list
    - load single alarm
      - countdown
      - no countdown
      - countdown cancelled
    - load multiple alarms
      - ?
    - reload alarms 
      - ok
      - not ok, cache
      - not ok, no cache
    - timed reload alarms
      - ok
      - not ok, cache
      - not ok, no cache
