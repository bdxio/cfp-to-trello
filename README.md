# cfp-to-trello

> Simple frontend application to export BDX I/O CFP proposals to Trello boards and cards for final ballot.

## Install

You first need to download manually the JSON file containing all the proposals and their votes from the CFP.
Log in to the CFP, go the URL <url cfp>/cfpadmin/allvotesJson and download the JSON result.
Put the file in the [data](./data) directory and name it `allvotesJson.json`  
**⚠️ DO NOT COMMIT THIS FILE EVER**

You can now install the dependencies using [yarn]() :
```bash
yarn
```

## Usage

Run [parcel](https://parceljs.org/) to serve the page :
```bash
yarn start
```

You then need to fill the correct organization name.  
Look for it in your Trello account, it should be in the URL when you select the organization which should contain the created boards.

You can then click on "Import" and wait for the magic to happen 🧙‍♀️ !  
It should take from 3 to 5 minutes, depending on the number of proposals to import.

## Contribute

PRs accepted.

## License

MIT © Benoît Giraudou