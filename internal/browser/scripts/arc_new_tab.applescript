tell application "Arc"
    if (count windows) = 0 then
        make new window
    end if
    tell front window
        make new tab with properties {URL: "PLACEHOLDER_URL"}
    end tell
    activate
end tell
