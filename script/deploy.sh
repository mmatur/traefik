#!/usr/bin/env bash
set -ex

if [ -n "${VERSION}" ]; then
  echo "Deploying..."
else
  echo "Skipping deploy"
  exit 0
fi

git config --global user.email "${TRAEFIKER_EMAIL}"
git config --global user.name "Traefiker"

# update traefik-library-image repo (official Docker image)
echo "Updating traefik-library-imag repo..."
git clone https://${GITHUB_TOKEN}@github.com/mmatur/traefik-library-image.git
cd traefik-library-image
./updatev2.sh "${VERSION}"
git add -A
echo "${VERSION}" | git commit --file -
echo "${VERSION}" | git tag -a "${VERSION}" --file -
git push -q --follow-tags -u origin master > /dev/null 2>&1

cd ..
rm -Rf traefik-library-image/

echo "Deployed"
