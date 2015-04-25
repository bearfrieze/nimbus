# Nimbus

Nimbus is the data back-end for [Litenin](https://github.com/bearfrieze/litenin) and is served through [Static](https://github.com/bearfrieze/static).

## Motivation

Litenin was originally backed by the [Google Feeds API](https://developers.google.com/feed/), but the API had some major drawbacks:

- Didn't pass along GUID's from feeds.
- Need for wrapper in order to abstract away complex API.
- Insufficient polling frequencies for regularly updated feeds.
- No support for batch requests of queries.

## Goal

The goal of Nimbus is to have none of the drawbacks of the Google Feeds API and keep feeds up to date in a stable and reliable fashion. Reliablity is considered more important than speed.
