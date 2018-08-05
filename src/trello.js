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

export const getMe = () => {
    return new Promise((resolve, reject) => {
        window.Trello.members.get("me", resolve, reject);
    });
};

export const createTrelloBoard = (name, idOrganization) => {
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

export const createTrelloList = (name, board) => {
    const list = {
        name,
        idBoard: board.id
    };

    return new Promise((resolve, reject) => {
        window.Trello.post("lists", list, resolve, reject);
    })
};

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