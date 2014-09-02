package main

const INSTALL_SHELL_SCRIPT = `#!/bin/bash
set -e
cd /Applications
rm -rf GitHubNotify.app
mkdir GitHubNotify.app
cd GitHubNotify.app
mkdir -p Contents/Resources/GitHubNotify.iconset
cd Contents/Resources/GitHubNotify.iconset
curl -LOs "https://octodex.github.com/images/labtocat.png"
convert labtocat.png -transparent white labtocat_t.png 2>/dev/null
mv labtocat_t.png labtocat.png
sips -p 1024 1024 "labtocat.png" >/dev/null
sips -z 16 16   "labtocat.png" --out "icon_16x16.png"    >/dev/null
sips -z 32 32   "labtocat.png" --out "icon_32x32.png"    >/dev/null
sips -z 128 128 "labtocat.png" --out "icon_128x128.png"  >/dev/null
sips -z 256 256 "labtocat.png" --out "icon_256x256.png"  >/dev/null
sips -z 512 512 "labtocat.png" --out "icon_512x512.png"  >/dev/null
cp   "icon_32x32.png"   "icon_16x16@2x.png"
sips -z 64 64   "labtocat.png" --out "icon_32x32@2x.png" >/dev/null
cp   "icon_256x256.png" "icon_128x128@2x.png"
cp   "icon_512x512.png" "icon_256x256@2x.png"
mv   "labtocat.png"  "icon_512x512@2x.png"
cd ..
iconutil -c icns "GitHubNotify.iconset" >/dev/null
rm -rf "GitHubNotify.iconset"
cd ..
cat > "Info.plist" <<FILE
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" \
"http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
<key>CFBundleExecutable</key>
<string>GitHubNotify</string>
<key>CFBundleIconFile</key>
<string>GitHubNotify</string>
</dict>
</plist>
FILE
mkdir -p MacOS
cat > "MacOS/GitHubNotify" <<'EOF'
#!/bin/bash
osascript -e 'tell application "Terminal" to do script "github-notify"'

EOF
chmod +x "MacOS/GitHubNotify"
defaults write com.apple.dock persistent-apps -array-add "<dict>
  <key>tile-data</key>
  <dict>
    <key>file-data</key>
    <dict>
      <key>_CFURLString</key>
      <string>/Applications/GitHubNotify.app</string>
      <key>_CFURLStringType</key>
      <integer>0</integer>
    </dict>
  </dict>
</dict>"
sleep 2
killall Dock
`
