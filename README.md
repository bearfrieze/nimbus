# Nimbus

Nimbus is the back-end powering [Litenin](https://github.com/bearfrieze/litenin).

## Motivation

The initial prototype of [Litenin](https://github.com/bearfrieze/litenin) was powered by the [Google Feeds API](https://developers.google.com/feed/). This allowed for swift prototyping, but unfortunately the API has some major drawbacks:

- Doesn't pass along GUID's from feeds.
- Needs wrapper in order to abstract away complex API.
- Insufficient polling frequencies for regularly updated feeds.
- No support for batch feed requests.

## Goal

The goal of Nimbus is to have none of the drawbacks of the Google Feeds API, while fulfilling the following two roles:

- Keep feeds up to date in a reliable fashion. Reliablity and consistency are the main goals while speed is a commodity.
- Respond quickly to batch feed requests. Speed and stability are the main goals.

## Concept

Nimbus stores feed information in a PostgreSQL database and maintains shallow JSON representations of feeds in a Redis cache. When handling batch feed requests Nimbus compiles cache hits to a single JSON array of feeds and adds any missing feeds to the polling queue afterwards.
