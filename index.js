const authenticationSuccess = function() {
    console.log('Successful authentication');
};

const authenticationFailure = function() {
    console.log('Failed authentication');
};

const authorize = () => {
    window.Trello.authorize({
        type: "popup",
        name: "CFP to Trello",
        scope: {
            read: true,
            write: true
        },
        success: authenticationSuccess,
        error: authenticationFailure
  });
};

window.onload = () => {
    document.getElementById("btn-import").addEventListener("click", authorize);
}
