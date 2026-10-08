# slk-plus: notification levels, stars, followed threads, avatars in the
# mention picker. Recorded against the demo workspaces.
Output docs/assets/demo/plus.gif
Source docs/assets/demo/plus-settings.tape

Hide
Type "./bin/slk --demo plus"
Enter
Sleep 1.5s
Show

# Jump to #deploys with the finder
Sleep 800ms
Ctrl+T
Sleep 400ms
Type@120ms "deploys"
Sleep 400ms
Enter
Sleep 1.2s

# m cycles the channel's Slack notification level
Type "m"
Sleep 1.6s
Type "m"
Sleep 1.6s

# s stars the channel
Type "s"
Sleep 1.6s

# Open Sam's thread and follow it with f; Jonas replies
Type@400ms "kk"
Sleep 400ms
Enter
Sleep 1.2s
Type "f"
Sleep 3s
Escape
Sleep 800ms

# @ in the composer: avatars in the mention picker
Type "i"
Sleep 400ms
Type@150ms "cc @"
Sleep 1.8s
Type@200ms "pr"
Sleep 1.5s
Escape
Sleep 300ms
Escape
Sleep 1s
