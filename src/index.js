import "babel-polyfill";

import { groupBy, keys, splitEvery } from "ramda";

import allvotesJson from "../data/allvotesJson.json";
import {
  authorize,
  getMe,
  getOrganization,
  createTrelloBoard,
  createTrelloList,
  createTrelloLabel,
  createTrelloCard
} from "./trello";

const BOARD_NAME_PREFIX = "Délibération";

const BOARDS_TO_CREATE = [
  {
    name: "Conférences",
    talkType: "conf"
  },
  {
    name: "LTs",
    talkType: "quick"
  },
  {
    name: "Hands-on",
    talkType: "lab"
  }
];

const AUDIENCE_LEVELS = {
  l1: "Pour tous niveaux",
  l2: "Intermédiaire",
  l3: "Expert"
};
const LANGS = { fr: "🇫🇷", en: "🇬🇧" };

/**
 * Function created when the document is loaded to display import progression.
 */
let log;

/**
 * Function clearing the previous import progression.
 */
let clearLog;

/**
 * Main function executed when the "Import" button is clicked.
 */
const run = async () => {
  clearLog();
  performance.mark("trello-import-start");
  const txtOutput = document.getElementById("txt-output");
  const scrollTextArea = textarea => {
    textarea.scrollTop = textarea.scrollHeight;
  };
  const scrollTextAreaInterval = setInterval(scrollTextArea, 50, txtOutput);

  const organizationName = document.getElementById("txt-organization").value;
  if (organizationName == null || organizationName.trim() == "") {
    window.alert("The name of the Trello organization is required");
    return;
  }

  try {
    const proposals = parseProposalsVotes(allvotesJson);
    log(`There are ${proposals.length} proposals to import...`);

    await authorize();
    const me = await getMe();
    log(`Successfully authenticated into Trello with user ${me.fullName}`);

    const organization = await getOrganization(organizationName);
    log(
      `Deliberation boards will be created for the organization ${
        organization.displayName
      }...`
    );

    const boards = await createDeliberationBoards(proposals, organization);
    performance.mark("trello-import-end");
    performance.measure("trello-import", "trello-import-start");
    const measure = performance.getEntriesByName("trello-import")[0];
    log(
      `${proposals.length} proposals successfully imported in Trello in ${
        measure.duration
      } ms 😎🚀🍾`
    );
    scrollTextArea(txtOutput);
  } catch (e) {
    console.error(e);
    log("An error occured, please check the console for more details");
  } finally {
    performance.clearMarks();
    performance.clearMeasures();
    clearInterval(scrollTextAreaInterval);
  }
};

/**
 * Parses the list of proposal votes to retain only the information needed to create the cards in Trello.
 *
 * @param {Array[proposalsVotes]} proposalsVotes The list of proposals associated to their votes given by the CFP application
 */
const parseProposalsVotes = proposalsVotes => {
  // Transforms a speaker object in a string containing his full name and his company
  const mapSpeaker = speaker => {
    let speakerLabel = `${speaker.firstName} ${speaker.name}`;
    // Speaker from the Gironde area should be identified clearly, a glass of wine should do the trick
    if (speaker.zipCode.startsWith("33")) speakerLabel += " 🍷";
    if (speaker.company) speakerLabel += ` (${speaker.company})`;

    return speakerLabel;
  };

  // Function computing the standard deviation (it seems the one given by the CFP is wrong)
  const computeStdDeviation = (average, voters) => {
    // We transform the voters in score and we filter out scores equals to 0 as they represent an absention
    const scores = voters.map(voter => voter.score).filter(score => score > 0);
    const squaredDifferences = scores
      .map(score => score - average)
      .map(n => n * n);
    const sumSquaredDifferences = squaredDifferences.reduce(
      (sum, n) => sum + n
    );
    const squaredDifferencesAverage = sumSquaredDifferences / scores.length;

    return Math.sqrt(squaredDifferencesAverage);
  };

  return proposalsVotes.map(({ proposal, votes }) => ({
    id: proposal.id,
    title: proposal.title,
    talkType: proposal.talkType,
    track: proposal.track,
    summary: proposal.summary,
    audienceLevel: AUDIENCE_LEVELS[proposal.audienceLevel],
    lang: LANGS[proposal.lang],
    speakers: proposal.allSpeakers.map(mapSpeaker).join(" / "),
    average: votes.average,
    // average has 3 decimal places so we do the same for the standard deviation
    stdDeviation: computeStdDeviation(votes.average, votes.voters).toFixed(3)
  }));
};

/**
 * Creates all deliberation boards in Trello and returns them as an array.
 *
 * @param {Array[Proposal]} proposals The list of all the proposals
 * @param organization The organization for which to create the boards
 * @returns {Array[TrelloBoard]} The list of Trello boards created
 */
const createDeliberationBoards = async (proposals, organization) => {
  const year = new Date().getFullYear();

  for (const { name, talkType } of BOARDS_TO_CREATE) {
    await createBoard(proposals, name, talkType, year, organization);
  }
};

/**
 * Creates the Trello board for all proposals of a given type and returns it.
 *
 * @param {Array<Proposal>} proposals The list of all proposals
 * @param {String} name The name of the board to create
 * @param {"conf" | "quick" | "lab"} talkType The type of the talk for which to create the board
 * @param {Number} year The year of the conference (usually the current year)
 * @param {String} organization The origanization in which the board should be created
 * @returns {TrelloBoard} The Trello board created
 */
const createBoard = async (proposals, name, talkType, year, organization) => {
  // We keep only proposal of the given type
  const boardProposals = proposals.filter(
    proposal => proposal.talkType === talkType
  );

  const boardName = `${BOARD_NAME_PREFIX} ${year} - ${name}`;
  log(
    `Creating the board ${boardName} for ${boardProposals.length} proposals...`
  );
  const board = await createTrelloBoard(boardName, organization);
  log("Creating lists for the board...");
  await createTrelloList("Sélection", board);
  await createTrelloList("Désistements", board);
  await createTrelloList("Backups Acceptés", board);
  await createTrelloList("Backups", board);
  await createDeliberationLists(board, boardProposals);
  await createTrelloList("Refusés", board);

  return board;
};

/**
 * Creates the deliberation lists for the proposals of one Trello board.
 *
 * @param {TrelloBoard} board The Trello board that should contain the created deliberation lists
 * @param {Array[Proposal]} proposals The proposals to add to the Trello board
 */
const createDeliberationLists = async (board, proposals) => {
  const proposalsByTrack = groupBy(proposal => proposal.track, proposals);
  const tracks = keys(proposalsByTrack).sort();

  const lastThirdProposals = [];
  for (const track of tracks) {
    const remainingProposals = await createTrackDeliberationLists(
      board,
      track,
      proposalsByTrack[track]
    );
    if (remainingProposals && remainingProposals.length > 0) {
      lastThirdProposals.push(...remainingProposals);
    }
  }

  await createDeliberationList(board, "T3", lastThirdProposals);
};

/**
 * Creates the Trello lists for a given track.
 * Two Trello lists are created, in which are added the two-thirds best proposals, the last third are returned to be put in a common Trello list with proposals from the other tracks.
 *
 * @param {TrelloBoard} board The board hosting the deliberation lists
 * @param {String} track The track of the proposals
 * @param {Array[Proposal]} proposals The list of the proposals for the track
 */
const createTrackDeliberationLists = async (board, track, proposals) => {
  const sortedProposals = proposals.sort((p1, p2) => p2.average - p1.average);
  // Proposals are splitted into three lists
  const thirdSize = Math.ceil(sortedProposals.length / 3);
  const sortedProposalsSplitted = splitEvery(thirdSize, sortedProposals);

  await createDeliberationList(
    board,
    `${track} - T1`,
    sortedProposalsSplitted[0]
  );
  await createDeliberationList(
    board,
    `${track} - T2`,
    sortedProposalsSplitted[1]
  );

  return sortedProposalsSplitted[2];
};

/**
 * Create the Trello deliberation list for a given third.
 *
 * @param {TrelloBoard} board The board in which the list should be created
 * @param {String} name The name of the list to create
 * @param {Array[Proposal]} proposals The list of proposals to add to the newly created list
 */
const createDeliberationList = async (board, name, proposals) => {
  log(
    `Creating "${name} deliberation list" for ${proposals.length} proposals...`
  );
  const deliberationList = await createTrelloList(name, board);
  for (const proposal of proposals) {
    await createProposalCard(deliberationList, proposal, board);
  }
};

/**
 * Creates a Trello card from a proposal into the given Trello list.
 *
 * @param {TrelloList} list The Trello list in which the Trello card should be added
 * @param {Proposal} proposal The proposal to create as a Trello card
 * @param {TrelloBoard} board The Trello board of the list (used to create labels in it)
 */
const createProposalCard = async (list, proposal, board) => {
  log(`Creating card for proposal ${proposal.title}...`);

  const idLabels = [];
  idLabels.push(await createLabel(proposal.track, board, "green"));
  idLabels.push(await createLabel(`avg:${proposal.average}`, board, "orange"));
  idLabels.push(
    await createLabel(`dev:${proposal.stdDeviation}`, board, "red")
  );
  idLabels.push(await createLabel(proposal.speakers, board, "purple"));
  idLabels.push(await createLabel(proposal.audienceLevel, board, "sky"));
  idLabels.push(await createLabel(proposal.lang, board, "pink"));

  await createTrelloCard(proposal.title, proposal.summary, list, idLabels);
};

/**
 * Trello labels must be created before used for a card.
 * As some cards may have some labels in common we cache them to not recreate identical Trello labels
 */
const trelloLabels = new Map();

/**
 * Creates a label and returns its Trello id.
 * The color defines the order of the associated label when displayed, a specific color will always be displayed before another one (for example a green label will always be displayed first). See also https://help.trello.com/article/797-adding-labels-to-cards for more information.
 *
 * @param {String} name The name of the label
 * @param {TrelloBoard} board The board to create the label for
 * @param {String} color The color of the label (might be one of yellow, purple, blue, red, green, orange, black, sky, pink, lime, null)
 */
const createLabel = async (name, board, color) => {
  const key = `${name}-${board.id}-${color}`;
  const idLabel = trelloLabels.get(key);
  if (idLabel == undefined) {
    const trelloLabel = await createTrelloLabel(name, board, color);
    trelloLabels.set(key, trelloLabel.id);
    return trelloLabel.id;
  }

  return idLabel;
};

window.onload = () => {
  const txtOutput = document.getElementById("txt-output");
  clearLog = () => {
    txtOutput.value = "";
  };
  log = msg => {
    txtOutput.value += `${msg}\n`;
  };

  document.getElementById("btn-import").addEventListener("click", run);
};
