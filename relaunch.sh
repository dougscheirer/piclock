#!/bin/bash

# auto-reload piclock for git source

cd /home/pi/go

function git_checkme
{
  SDIR=$1
  OPTS="--git-dir $SDIR/.git --work-tree $SDIR"
  REMOTE=$(git $OPTS ls-remote origin refs/heads/master | cut -f 1)
  if git $OPTS cat-file -e $REMOTE ; then
    echo "Up to date" > /dev/null
    return 0
  else
    echo "Remote has changed"
    return 1
  fi
}

git_checkme src/piclock
if [ $? -eq 0 ];
  return 0
else
  pushd src/piclock
  git pull origin master
  if [ ! $? -eq 0 ]; then
    return 1
  fi
  popd
  go install piclock
  if [ ! $? -eq 0 ]; then
    return 1
  fi
  killall piclock
  bin/piclock
  return $?
fi
