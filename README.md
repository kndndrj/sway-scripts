# Sway Scripts

This repository contains some of my scripts that I use with my sway config.

## `sway-reflex`

This one is basically `autotiling` on steroids.
You provide a physical window size and the script organizes windows by itself.
Useful for large monitors, where a single window over the whole screen is just too big.

example:
```
# [.config/sway/config]

exec_always sway-reflex -window_size 500x300 -default_gaps 20
```

## `sway-scratch`

This starts a server to manage scratchpads - this server can then be controlled via
subsequent commands.

example:
```
# [.config/sway/config]

# start the server
exec_always sway-scratch serve

# bind scratchpads to keys
bindsym $mod+d exec sway-scratch call kitty -position left
bindsym $mod+f exec sway-scratch call kitty -position right
```
