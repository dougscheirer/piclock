#!/bin/bash

die()
{
  echo "$*"
  exit 1
}

NORESTART=$1

GO=/home/pi/gvm/go

echo "go: $($GO version)"

# do a git pull, a go test/build, and start the service
su pi -c "git pull" || die "failed to perform git pull"

# always rebuild the pireload binary
pushd pireload
su pi -c "$GO build" || die "Failed to rebuild pireload"
popd

# is the sha differnt than the build version?
su pi -c "git diff-index --quiet HEAD"
if [ $? -eq 0 ] ; then 
  SHA=$(git rev-parse HEAD)
  NOW=$(./piclock -version)
else
  SHA="contains uncommited changes"
  NOW="who cares"
fi
if [ "$NOW" != "Version $SHA" ] ; then 
  # generate a new version file from the template
  sed 's/unknown/'"$SHA"'/' versionInfo.go.tmpl > versionInfo.go

  su pi -c "$GO test -v" >> ./test-output.txt 2>&1
  
  if [ ! $? -eq 0 ] ; then 
    die "go tests failed!"
  fi

  su pi -c "$GO build" || die "go build failed!"

  # revert the go file we changed
  su pi -c "git co -- versionInfo.go"
fi

if [ "$NORESTART" == "" ] ; then
  systemctl daemon-reload
  systemctl restart piclock.service || die "failed to restart piclock"
  # this will exit the script as we are launched by it
  systemctl restart pireload.service || die "failed to restart pireload"
if
