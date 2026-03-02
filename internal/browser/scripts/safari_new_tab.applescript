tell application "Safari"
    if (count windows) = 0 then
        make new document
    end if
    tell front window
        set newTab to make new tab at end of tabs
        set URL of newTab to "PLACEHOLDER_URL"
        set current tab to newTab
    end tell
    activate
end tell
