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
            error: reject,
        })
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
export const getOrganization = (name) => {
    return new Promise((resolve, reject) => {
        window.Trello.organizations.get(name, resolve, reject);
    });
};

/**
 * Creates a board for an organization.
 * 
 * @param {String} name The name of the board to create
 * @param {TrelloOrganization} organization The organization in which the board should be created
 */
export const createTrelloBoard = (name, organization) => {
    const board = {
        name,
        defaultLabels: false,
        defaultLists: false,
        idOrganization
    };

    return new Promise((resolve, reject) => {
        window.Trello.post("boards", board, resolve, reject);
    });
};

/**
 * Creates a list in a board.
 * 
 * @param {String} name The name of the list to create
 * @param {TrelloBoard} board The board in which the list should be created
 */
export const createTrelloList = (name, board) => {
    const list = {
        name,
        idBoard: board.id
    };

    return new Promise((resolve, reject) => {
        window.Trello.post("lists", list, resolve, reject);
    })
};

/**
 * Creates a Trello label for a given board
 * @param {String} name The name of the label (which will be displayed on the card)
 * @param {TrelloBoard} board The board to create the label for
 * @param {String} color The color of the label (might be one of yellow, purple, blue, red, green, orange, black, sky, pink, lime, null)
 */
export const createTrelloLabel = (name, board, color) => {
    const label = {
        idBoard: board.id,
        name,
        color,
    };

    return new Promise((resolve, reject) => {
        window.Trello.post("labels", label, resolve, reject);
    });
};

/**
 * Creates a Trello card
 * @param {String} name The name of the card
 * @param {String} description The description of the card
 * @param {TrelloList} list The list in which the card should be created
 * @param {Array[String]} labels List of id labels to add to the card
 */
export const createTrelloCard = (name, description, list, labels) => {
    const card = {
        name,
        description,
        idList: list.id,
        idLabels: labels.join(","),
    };

    return new Promise((resolve, reject) => {
        window.Trello.post("cards", card, resolve, reject);
    });
};