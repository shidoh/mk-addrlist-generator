# RouterOS fetcher for mk-addrlist-generator
# https://github.com/shidoh/mk-addrlist-generator
#
# Downloads one generated list and imports it, with retry and error reporting.
# Install one copy per list, then drive it from the scheduler:
#
#   /system/script/add name=fetch-blocklist source=[/file/get \
#       [find name="fetch-list.rsc"] contents]
#   /system/scheduler/add name=fetch-blocklist start-time=startup interval=2h \
#       on-event="/system/script/run fetch-blocklist"
#
# Only the first three values need changing.

:local name "mk-addrlist"
:local url "http://server:8080/list/blocklist"
:local fileName "mk-addrlist-blocklist.rsc"

# Retry window is maxAttempts * retryDelay. The default of ten minutes covers a
# router that finishes booting before the network the source lives on is up.
:local maxAttempts 20
:local retryDelay 30s

:log info "$name starting address list update"

# Drop any leftover from a previous run. If that run died during import the file
# survives, and a later failed fetch would silently re-import a stale list.
:onerror err { /file remove [/file find name=$fileName] } do={}

# A bare /tool fetch RAISES on an unreachable source and ABORTS the script, so
# error handling placed after it never runs. That is why the fetch is wrapped
# rather than followed by an :if on the downloaded file.
:local fetched false
:for i from=1 to=$maxAttempts do={
    :if ($fetched = false) do={
        :onerror err {
            /tool fetch url="$url" mode=http dst-path=$fileName
            :set fetched true
        } do={
            :log warning "$name fetch attempt $i/$maxAttempts failed: $err"
            :if ($i < $maxAttempts) do={ :delay $retryDelay }
        }
    }
}

:if ($fetched = false) do={
    :log error "$name giving up, $url unreachable after $maxAttempts attempts"
} else={
    :if ([:len [/file find name=$fileName]] = 0) do={
        :log error "$name fetch succeeded but $fileName is missing"
    } else={
        :log info "$name importing address list"
        # The generated script clears the list before repopulating it, so a
        # failed import leaves the previous contents untouched.
        :onerror err {
            /import file-name=$fileName
            :log info "$name address list update completed"
        } do={
            :log error "$name import failed: $err"
        }
        :onerror err { /file remove [/file find name=$fileName] } do={}
    }
}
