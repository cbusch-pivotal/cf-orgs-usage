#!/bin/bash
set -e

function pause(){
   echo " "
   read -p "$*"
}

echo "----"
echo "Before proceeding, make sure to edit and run the 'auditor.sh' script"
echo "in the root folder to create the Auditor Administrator user to be"
echo "used by cf-orgs-uage app."
pause "Press <enter> to continue or Ctrl-C to exit..."

# copy files for src
cd resources

echo " " 
echo "Copying source files for tile inclusion..."
# zip up the files
#rm -f cf-orgs-usage.zip
#zip -r cf-orgs-usage.zip ../../*.go ../../manifest.yml ../../glide.yaml ../../vendor/*

# OR copy the code - edit tile.yml to reference which method
rm -r go-code || true
mkdir go-code
cp ../../*.go go-code
cp ../../manifest.yml go-code
cp ../../glide.yaml go-code
cp -r ../../vendor go-code

ls -hal go-code

cd ..

pause "If the copy was successful, hit <enter>. Otherwise hit Ctrl-C..."

# now build the tile
echo " "
echo "Building the tile..."
tile build

pause "Was the build successful? If so, hit <enter>, otherwise Ctrl-C..."

# import and install the tile
echo " "
echo "Importing the new tile..."
pcf import `ls product/cf-orgs-usage*`

echo " "
echo "Installing the new tile..."
pcf install cf-orgs-usage `cat tile-history.yml | grep version | cut -f2 -d " "`

pause "Were the Import and Install successful? Ctrl-C if not..."

pause "In you need to alter the tile settings, please go to the Operations Manager GUI to make updates. When done, return here and hit the <enter> key."

# apply the install
echo " "
echo "Applying changes in PCF. This will take a while..."
pcf apply-changes

