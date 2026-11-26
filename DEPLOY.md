# IstkharMusic deploy

All variables are in `sample.env`. Required: `API_ID API_HASH TOKEN STRING1 MONGO_URI OWNER_ID LOGGER_ID API_KEY`.
`API_URL`, `DB_NAME=Anon`, support links and images already have defaults.

## Heroku (container stack)
```
heroku create <app> --stack=container
heroku config:set API_ID=.. API_HASH=.. TOKEN=.. STRING1=.. MONGO_URI=.. OWNER_ID=.. LOGGER_ID=.. API_KEY=..
git push heroku HEAD:main
```
Or use the Deploy to Heroku button in README.md.
```
heroku ps:scale worker=1
```
(`heroku.yml` builds the Dockerfile as a worker.)

## Railway
New project -> Deploy from repo (uses `railway.json` + Dockerfile) -> add the variables above.

## VPS
```
cp sample.env .env && nano .env
./deploy/vps.sh
```

## Extra
* BotFather: `/setprivacy` -> Disable, otherwise in groups with many bots plain `/play` is not delivered. Making the bot admin also works.
* Add your own `PING_IMG` and `SUPPORT_IMG` (URL or file path).
