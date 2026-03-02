tell application "BROWSER_APP"
    if (count windows) = 0 then
        make new window
    end if
    tell front window
        make new tab with properties {URL: "PLACEHOLDER_URL"}
        set active tab index to (count tabs)
    end tell
    activate
end tell
