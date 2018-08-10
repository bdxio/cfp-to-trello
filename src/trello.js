/**
 * Asks the user to authorize the API to use his Trello account.
 */
export const authorize = () => {
  return new Promise((resolve, reject) => {
    window.Trello.authorize({
      type: "popup",
      name: "CFP to Trello",
      scope: {
        read: true,
        write: true
      },
      success: resolve,
      error: reject
    });
  });
};

/**
 * Returns the current authenticated user.
 */
export const getMe = () => {
  return new Promise((resolve, reject) => {
    window.Trello.members.get("me", resolve, reject);
  });
};

/**
 * Returns the organization given its name (https://developers.trello.com/reference#organization-object).
 * This name can be found in the URL used by Trello (it might not be the same as the one displayed in Trello ; an organization named "Test" won't have this name in the Trello API because it is very likey that other organizations with the very same name exist in other Trello accounts).
 *
 * @param {String} name The "technical" name of the organization
 */
export const getOrganization = name => {
  return new Promise((resolve, reject) => {
    window.Trello.organizations.get(name, resolve, reject);
  });
};

/**
 * Creates a board for an organization.
 *
 * @param {String} name The name of the board to create
 * @param {TrelloOrganization} organization The organization in which the board should be created
 * @param {"private" | "org" | "public"} The visibility of the board, private by default
 */
export const createTrelloBoard = (name, { id: idOrganization }, visibility = "private") => {
  const board = {
    name,
    idOrganization,
    defaultLabels: false,
    defaultLists: false,
    prefs_permissionLevel: visibility
  };

  return new Promise((resolve, reject) => {
    window.Trello.post("boards", board, resolve, reject);
  });
};

/**
 * Map containing for each board id the number of list created, this allow to use an incrementing position when
 * creating a new list to keep them displayed in their creation order.
 */
const boardNbLists = new Map();

/**
 * Creates a list in a board.
 *
 * @param {String} name The name of the list to create
 * @param {TrelloBoard} board The board in which the list should be created
 */
export const createTrelloList = (name, { id: idBoard }) => {
  const nbLists = boardNbLists.get(idBoard);
  const pos = nbLists || 0;
  boardNbLists.set(idBoard, pos + 1);

  const list = {
    name,
    idBoard,
    // A totally arbitrary computation found empirically to have the lists displayed in their creation order
    pos: 2048 + pos * 2048
  };

  return new Promise((resolve, reject) => {
    window.Trello.post("lists", list, resolve, reject);
  });
};

/**
 * Creates a Trello label for a given board.
 * 
 * @param {String} name The name of the label (which will be displayed on the card)
 * @param {TrelloBoard} board The board to create the label for
 * @param {String} color The color of the label (might be one of yellow, purple, blue, red, green, orange, black, sky, pink, lime, null)
 */
export const createTrelloLabel = (name, { id: idBoard }, color) => {
  const label = {
    idBoard,
    name,
    color
  };

  return new Promise((resolve, reject) => {
    window.Trello.post("labels", label, resolve, reject);
  });
};

/**
 * Creates a Trello card.
 * 
 * @param {String} name The name of the card
 * @param {String} desc The description of the card
 * @param {TrelloList} list The list in which the card should be created
 * @param {Array[String]} labels List of id labels to add to the card
 */
export const createTrelloCard = (name, desc, { id: idList }, labels) => {
  const card = {
    name,
    desc,
    idList,
    idLabels: labels.join(",")
  };

  return new Promise((resolve, reject) => {
    window.Trello.post("cards", card, resolve, reject);
  });
};

/**
 * Create a comment on a card.
 * 
 * @param {String} comment The comment to add to the card
 * @param {TrelloCard} card The card on which the comment should be added
 */
export const createTrelloComment = (comment, card) => {
  return new Promise((resolve, reject) => {
    window.Trello.post(`cards/${card.id}/actions/comments`, { text: comment }, resolve, reject);
  });
};