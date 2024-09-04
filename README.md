# laundry-notify

A simple golang service that lets users subscribe to notifications when their laundry is done via ntfy.sh

## Why?

Using a smart plug lets you get notifications when your appliances are turned on or off (via homeassistant, or some other service), but if you live in a house with multiple people you will end up being notified for everyone's laundry, not just your own.

This service lets you subscribe to be notified for specific runs of an appliance, so that you only get a notification for your own laundry.

## Installation

Before you get started, you will need an mqtt broker running somewhere, and some way of sending events from your appliances to mqtt. I use a home assistant automation to do this, but anything that lets you send messages via mqtt based on the power state of the appliances will do the trick.

You will need to set up mqtt to receive events on the topic you specify in the config, with either `started_at=<timestamp>` or `finished_at=<timestamp>`. Please ensure the timestamps are using ISO 8601 format for compatibility.

This repo contains a dockerfile you can use to build a docker container.

Building the go binary should also work, as all the templates are embedded into it.

## Running

The following env vars are required:

| Variable            | Value                    |Notes
|---------------------|--------------------------|-----
| DB_DSN              | data/data.db             |The location of the sqlite database
| MQTT_URL            | mqtt://10.0.0.3:1883     |MQTT broker url
| MQTT_CLIENT_ID      | desktop                  |An id to identify the client to the mqtt broker
| MQTT_USERNAME       | username                 |The mqtt username
| MQTT_PASSWORD       | password                 |The mqtt password for the user
| MQTT_TOPIC          | notify/laundry/+         |The mqtt topic to listen for events on. Note that `+` means wildcard subtopic, so in this case, any topic under /laundry will be recieved
| NTFY_BASE_TOPIC     | BaseTopic                |The base ntfy topic. This will be the first part of the topic used on ntfy.sh, appended with the registered username. For example, if I register as 'user', the full nfty topic would be BaseTopic-user

These can be provided via docker (compose) env vars, or using a .env file.

## How does it work?

This service relies on events coming from mqtt. I use homeassistant to populate these events, but you could do it a different way if you prefer. The important thing is that the service listens to a specific topic for events with a `started_at` and `finished_at` payload, with the current UTC timestamp.

Users can input their name to receive notifications from finished events. Once registered, they will be redirected to a ntfy.sh channel, specifically for that user.

When an event comes in, the service will check to see which users have registered to receive a notification for it, and send any that have a notification on the ntfy channel matching their username.

This way you can subscribe to notifications on ntfy.sh for your username and only be notified for your own stuff

You can also use the home page to see if a load is currently in progress.

## Tech used

- Tailwindcss
- MQTT
- Sqlite
- ntfy.sh

This is based on the example implementation here: https://github.com/benbjohnson/wtf
