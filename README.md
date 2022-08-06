# cfp-to-trello

> Simple frontend application to export BDX I/O CFP proposals to Trello boards and cards for final ballot.

This application is used to import CFP proposals into Trello boards.

For each format of proposals (conference, quickie or hands on lab) a dedicated Trello board is created.  
All proposals are then grouped by categories and split into three tiers, from top-rated to lower rated.  
For each proposal a Trello card is created and put in a Trello list specific to its track and rating, for the first two 
tiers. All last tier proposals are put together in another Trello list.

## Compile

```shell
go build
```

## Usage

First you need to download the JSON export from the CFP.  
Log in to the CFP, go the proposals page and click on "Export...>JSON file".

You can then run the application:

```shell
./cfp-to-trello -trello-key <YOUR TRELLO KEY> -trello-secret <YOUR TRELLO SECRET> -event-id <YOUR EVENT ID> -json <PATH TO JSON>
```

Your Trello API key and secret can be found [there](https://trello.com/app-key).  
Create one if needed and use http://localhost:8000 as origin.

The event ID can be found in the event profile in Conference-Hall, it's the last part of the public event URL.

The creation of all elements in Trello might take some time (around 5 minutes for 350 proposals).

## Contribute

PRs accepted.

## License

MIT © BDX I/O Team
