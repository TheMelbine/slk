#!/bin/sh
# Builds slk-notifier.app and installs it to ~/Applications, where slk
# looks for it. The app's icon becomes the icon of slk's notifications:
# pass an .icns (or an .app to take its icon from) as $1. Without one the
# script uses Slack's icon when Slack.app is installed.
set -eu

here=$(cd "$(dirname "$0")" && pwd)
name=${SLK_NOTIFIER_NAME:-Slack}
icon=${1:-/Applications/Slack.app}
dest=${SLK_NOTIFIER_DEST:-$HOME/Applications}
app="$dest/slk-notifier.app"

if [ -d "$icon" ]; then
	file=$(defaults read "$icon/Contents/Info" CFBundleIconFile 2>/dev/null || true)
	case "$file" in "") icon="" ;; *.icns) icon="$icon/Contents/Resources/$file" ;; *) icon="$icon/Contents/Resources/$file.icns" ;; esac
fi

rm -rf "$app"
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"
swiftc -O "$here/main.swift" -o "$app/Contents/MacOS/slk-notifier"
if [ -n "$icon" ] && [ -f "$icon" ]; then
	cp "$icon" "$app/Contents/Resources/AppIcon.icns"
fi
sed "s/@NAME@/$name/g" "$here/Info.plist" >"$app/Contents/Info.plist"
codesign --force --sign - "$app"

# Notification Center caches app icons by bundle; re-register the bundle
# so a new icon shows up.
/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister -f "$app"

# The first launch through LaunchServices asks for notification permission.
open -W -n "$app" --args --title slk --body "Notifications from slk will look like this."
echo "installed $app"
