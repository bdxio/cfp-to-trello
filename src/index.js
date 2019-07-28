import "babel-polyfill";
import { groupBy, keys, splitEvery } from "ramda";
import {
  authorize,
  getMe,
  getOrganization,
  createTrelloBoard,
  createTrelloList,
  createTrelloLabel,
  createTrelloCard
} from "./trello";
import cfp from "../data/export.json";

const BOARD_NAME_PREFIX = "Délibération";

/**
 * Categories of talks (Design, Backend, Frontend...).
 */
const CATEGORIES = cfp.categories;

/**
 * Some talks don't have category set, this map is used to fix it.
 * Keys are talks title as the export doesn't include the id of the talk.
 */
const CATEGORIES_FIX = {
  "Programmation fonctionnelle facile avec elm": "Front-end",
  "Comment Elm a transformé mon expérience de développeur front-end": "Front-end",
  "Une bonne expérience utilisateur pour protéger vos données, c’est possible": "Design & UX",
  "COBOL et envahisseurs": "Back-end",
  "Créez votre première extension VS Code": "Découverte",
  "De Java à Go ": "Back-end",
  "Concourse, des pipelines CI/CD pour \"l'ère cloud native\"": "Cloud & DevOps",
  "Vers l'infini et au-delà avec Angular !": "Front-end",
};

/**
 * Formats of talks (Conférence, Quickie, Hands on lab...).
 */
const FORMATS = cfp.formats;

/**
 * Some talks don't have format set, this map is used to fix it.
 * Keys are talks title as the export doesn't include the id of the talk.
 */
const FORMATS_FIX = {
  "Programmation fonctionnelle facile avec elm": "Hands on lab",
  "Comment Elm a transformé mon expérience de développeur front-end": "Conférence",
  "Le cloud et le devops au profit de mon poste de développement.": "Conférence",
  "Des animations SVG en JS, cools et super rapides ? Bien sûr !": "Quickie",
  "JAMstack, ou comment faire des sites statiques modernes et rapides": "Quickie",
  "Améliorez votre façon de taper du code au quotidien": "Quickie",
  "Commencez à bloguer dès aujourd'hui": "Quickie",
  "Using Kubeflow Pipelines for building machine learning pipelines": "Conférence",
  "Des conteneurs sans baleine": "Conférence",
  "Chaine de fabrication Web, du développement au monitoring en production": "Conférence",
  "Scripting en Go (15 min)": "Quickie",
  "Du puzzle aux Légo, le périple de nos architectures logicielles": "Conférence",
  "Du POC à la Prod, un projet de data science mis à nu ! ": "Conférence",
  "De Java à Go ": "Quickie",
  "DataScience from the trenches": "Conference",
  "Développement et productivité à l’ère des infras cloud native: l’approche Eclipse Che": "Conférence",

};

/**
 * List of speakers with at least one proposal.
 */
const SPEAKERS = cfp.speakers;

const AUDIENCE_LEVELS = {
  beginner: "Débutant",
  intermediate: "Intermédiaire",
  advanced: "Avancé"
};  

const LANGS = {
  France: "🇫🇷",
  French: "🇫🇷",
  Français: "🇫🇷",
  français: "🇫🇷",
  Francais: "🇫🇷",
  francais: "🇫🇷",
  Fançais: "🇫🇷",
  FR: "🇫🇷",
  fr: "🇫🇷",
  Fr: "🇫🇷",
  french: "🇫🇷",
  en: "🇬🇧",
  English: "🇬🇧",
  "English or French (any preferences?)": "🇫🇷/🇬🇧",
  "French (preferred) or English": "🇫🇷/🇬🇧",
  "Français / English?": "🇫🇷/🇬🇧",
  "French (but can be in english)": "🇫🇷/🇬🇧",
  "Français ou Anglais": "🇫🇷/🇬🇧",
  "Français ou English": "🇫🇷/🇬🇧",
  "FR or EN": "🇫🇷/🇬🇧",
  "English or French": "🇬🇧/🇫🇷",
  "Anglais (de préférence) ; Français (si nécessaire)": "🇬🇧/🇫🇷",
};

const CFP_URL = "https://conference-hall.io/organizer/event";
const CFP_EVENT_ID = "TODO";

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
  const organizationName = document.getElementById("txt-organization").value;
  if (organizationName == null || organizationName.trim() == "") {
    window.alert("The name of the Trello organization is required");
    return;
  }

  clearLog();
  performance.mark("trello-import-start");
  const txtOutput = document.getElementById("txt-output");
  const scrollTextArea = textarea => {
    textarea.scrollTop = textarea.scrollHeight;
  };
  const scrollTextAreaInterval = setInterval(scrollTextArea, 50, txtOutput);

  try {
    const proposals = await parseTalks(cfp.talks);
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

    createBoardsParagraph(boards);

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
 * Create an HTML paragraph used to display information on created boards.
 * 
 * @param {Array[TrelloBoard]} boards The created Trello boards list
 */
const createBoardsParagraph = boards => {
  const paragraph = document.createElement("p");
  paragraph.innerHTML = boards.map(createBoardUrlText).join("<br>");
  document.body.appendChild(paragraph);
}

/**
 * Create a text to display information on a created Trello board
 * 
 * @param {TrelloBoard} board The Trello board
 * @returns {TextNode} The text containing the information about the created board
 */
const createBoardUrlText = board => {
  return `Le board ${board.name} a bien été créé : <a href="${board.shortUrl}" target="_blank">${board.shortUrl}</a>`;
};

/**
 * Parses the list of talks to retain only the information needed to create the cards in Trello.
 *
 * @param {Array[proposalsVotes]} proposalsVotes The list of proposals associated to their votes given by the CFP application
 */
const parseTalks = async (talks) => {
  // Transforms a speaker in a string containing his full name and his company
  const parseSpeaker = async speakerUid => {
    const speaker = SPEAKERS.find(speaker => speaker.uid === speakerUid);
    let speakerLabel = `${speaker.displayName} -`;
    if (speaker.address && speaker.address.latLng) {
      try {
        const location = await findLocation(speaker.address);
        speakerLabel += ` ${location.city}`;
        // Speaker from the Gironde area should be identified clearly, a glass of wine should do the trick
        if (location.zipCode.startsWith("33")) speakerLabel += " 🍷";
      } catch (e) {
        console.error(e);
      }
    } else {
      speakerLabel += " 🗺️";
    }
    if (speaker.company) speakerLabel += ` (${speaker.company})`;

    return speakerLabel;
  };

  const parseSpeakers = async (speakers) => {
    return await Promise.all(speakers.map(speaker => parseSpeaker(speaker)));
  }

  const parseLanguage = ({ language }) => {
    // Language can be null or undefined, in this case we use FR as default
    const lang = language ? LANGS[language.trim()] : LANGS.FR;
    if (!lang) console.error(`${language} is not a known language!`);
    return lang ? lang : `🙊 (${language})`;
  };

  const parseCategory = ({ categories: categoryId, title: talkTitle }) => {
    if (!categoryId) {
      const categoryName = CATEGORIES_FIX[talkTitle];
      if (!categoryName) throw `A category fix is needed for talk "${talkTitle}"`;
      return categoryName;
    }
    const category = CATEGORIES.find(category => category.id === categoryId);
    return category ? category.name : "NO CATEGORY";
  }

  const parseFormat = ({ formats: formatId, title: talkTitle }) => {
    if (!formatId) {
      const formatName = FORMATS_FIX[talkTitle];
      if (!formatName) throw `A format fix is needed for talk "${talkTitle}"`;
      return formatName;
    }
    const format = FORMATS.find(format => format.id === formatId);
    return format ? format.name : "NO FORMAT";
  };

  const parseTalk = async (talk) => ({
    id: talk.uid,
    title: talk.title,
    category: parseCategory(talk),
    format: parseFormat(talk),
    abstract: talk.abstract,
    audienceLevel: AUDIENCE_LEVELS[talk.level],
    lang: parseLanguage(talk),
    speakers: (await parseSpeakers(talk.speakers)).join(" / "),
    privateMessage: talk.comments,
    average: Number(talk.rating).toFixed(2),
    loves: talk.loves,
    hates: talk.hates,
  });

  return await Promise.all(talks.map(parseTalk));
};

const findLocation = async ({ latLng: { lat, lng: lon } } = {} ) => {
  const request = `https://geo.api.gouv.fr/communes?lat=${lat}&lon=${lon}&fields=codesPostaux&format=json&geometry=centre`;
  const response = await fetch(request);
  if (response.ok) {
    const json = await response.json();
    if (json[0]) {
      return {
        city: json[0].nom,
        zipCode: json[0].codesPostaux[0],
      };
    } else {
      console.error(`no location found for coordinates ${lat},${lon}!`);
      return {
        city: "🗺️",
        zipCode: "00000",
      };
    }
  }
  throw `unable to find zip code for coordinates ${lat}/${lon}!`;
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
  const boards = [];

  for (const format of FORMATS) {
    if (format.name === "Hands on lab") boards.push(await createBoard(proposals, format, year, organization));
  }

  return boards;
};

/**
 * Creates the Trello board for all proposals of a given type and returns it.
 *
 * @param {Array<Proposal>} proposals The list of all proposals
 * @param format The format of the board to create (Quickie, Conférence or Hands on lab)
 * @param {String} format The format of the talk for which to create the board
 * @param {Number} year The year of the conference (usually the current year)
 * @param {String} organization The origanization in which the board should be created
 * @returns {TrelloBoard} The Trello board created
 */
const createBoard = async (proposals, format, year, organization) => {
  // We keep only proposal of the given type
  const boardProposals = proposals.filter(proposal => proposal.format === format.name);

  const boardName = `${BOARD_NAME_PREFIX} ${year} - ${format.name}`;
  log(
    `Creating the board ${boardName} for ${boardProposals.length} proposals...`
  );
  const board = await createTrelloBoard(boardName, organization, "org");
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
  const proposalsByCategory = groupBy(proposal => proposal.category, proposals);
  const categories = keys(proposalsByCategory).sort();

  const lastThirdProposals = [];
  for (const category of categories) {
    const remainingProposals = await createCategoryDeliberationLists(
      board,
      category,
      proposalsByCategory[category]
    );
    if (remainingProposals && remainingProposals.length > 0) {
      lastThirdProposals.push(...remainingProposals);
    }
  }

  await createDeliberationList(board, "T3", lastThirdProposals);
};

/**
 * Creates the Trello lists for a given category.
 * Two Trello lists are created, in which are added the two-thirds best proposals, the last third are returned to be put in a common Trello list with proposals from the other categories.
 *
 * @param {TrelloBoard} board The board hosting the deliberation lists
 * @param {String} category The category of the proposals
 * @param {Array[Proposal]} proposals The list of the proposals for the category
 */
const createCategoryDeliberationLists = async (board, category, proposals) => {
  const sortedProposals = proposals.sort((p1, p2) => p2.average - p1.average);
  // Proposals are splitted into three lists
  const thirdSize = Math.ceil(sortedProposals.length / 3);
  const sortedProposalsSplitted = splitEvery(thirdSize, sortedProposals);

  await createDeliberationList(
    board,
    `${category} - T1`,
    sortedProposalsSplitted[0]
  );

  if (sortedProposalsSplitted[1] && sortedProposalsSplitted[1].length > 0) {
    await createDeliberationList(
      board,
      `${category} - T2`,
      sortedProposalsSplitted[1]
    );
  }

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
  idLabels.push(await createLabel(proposal.category, board, "green"));
  idLabels.push(
    await createLabel(
      `avg: ${proposal.average}`,
      board,
      "orange"
    )
  );
  idLabels.push(
    await createLabel(`${proposal.loves} ❤️ / ${proposal.hates} ☠️`, board, "red")
  );
  idLabels.push(await createLabel(proposal.speakers, board, "purple"));
  idLabels.push(await createLabel(proposal.audienceLevel, board, "sky"));
  idLabels.push(await createLabel(proposal.lang, board, "pink"));

  const proposalUrl = `${CFP_URL}/cfpadmin/proposal/${proposal.id}`;
  const proposalLink = `[Proposal](${proposalUrl})`;
  const votesLink = `[Votes](${proposalUrl}/score)`;
  const approveLink = `[APPROVE](${CFP_URL}/ar/preaccept/${proposal.id})`;
  const cardDescription = `${proposalLink} • ${votesLink} • ⚠️ ${approveLink} ⚠️
  
  ---
  
  ${proposal.abstract}
  
  ---
  
  ${proposal.privateMessage}`;

  const card = await createTrelloCard(
    proposal.title,
    cardDescription,
    list,
    idLabels
  );
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
