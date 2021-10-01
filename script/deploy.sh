#!/usr/bin/env bash
set -e

if [ -n "$TRAVIS_TAG" ]; then
  echo "Deploying..."
else
  echo "Skipping deploy"
  exit 0
fi

git config --global user.email "$TRAEFIKER_EMAIL"
git config --global user.name "Traefiker"

# load ssh key
eval "$(ssh-agent -s)"
chmod 600 /home/semaphore/.ssh/traefiker_rsa_new
ssh-add /home/semaphore/.ssh/traefiker_rsa_new

# update traefik-library-image repo (official Docker image)
echo "Updating traefik-library-imag repo..."
git clone git@github.com:mmatur/traefik-library-image.git
cd traefik-library-image
./updatev1.sh $VERSION
git add -A
echo $VERSION | git commit --file -
echo $VERSION | git tag -a $VERSION --file -
git push -q --follow-tags -u origin master > /dev/null 2>&1

cd ..
rm -Rf traefik-library-image/

echo "Deployed"
