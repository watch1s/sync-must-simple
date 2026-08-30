# AMO Submission Notes for Reviewers

Hello Mozilla Review Team,

Thank you for reviewing **sync-must-simple**. 

## Purpose
This extension allows users to synchronize their reading/scroll position on long web pages (like webtoons, long articles, or documentation) across their own devices over a Local Area Network (LAN). It is paired with an open-source local Go server.

## Privacy & Permission Design (Why we use Optional Permissions)
To respect user privacy, this extension **does not** request broad `<all_urls>` host permissions by default. Instead, it heavily relies on `optional_permissions`. 

1. The user manually inputs the domain they want to sync in the extension's Options page.
2. We call `browser.permissions.request()` to request host permissions dynamically just for that specific domain.
3. Once granted, we use `browser.scripting.registerContentScripts()` to inject our scroll-tracking script *only* into the approved domains.

This ensures the extension only has access to the specific sites the user wants to sync, and nothing else.

## Testing the Extension
Because this extension requires a local server to function completely, you can test it by either:
1. Reviewing the source code (which is very minimal and clean).
2. Running the open-source Go server provided in the linked GitHub repository (instructions are in the README). You can run it locally on `127.0.0.1:8787`.

## Android Compatibility
This Manifest V3 extension is fully compatible with Firefox for Android. The options page and content scripts have been designed to work on mobile browsers seamlessly.
