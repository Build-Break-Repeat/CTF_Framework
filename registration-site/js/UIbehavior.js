//This will load the challenges when the page challenges page loads.
document.addEventListener("DOMContentLoaded", () => {
    if (document.getElementById("challenge-container")) {
        //this wil be called to load the challenges
        loadChallenges();
    }
});


//this function fetches the challenges from the JSON file.
async function loadChallenges() {
    try {
        //this is the location of the information
        const response = await fetch("../challenges.json");
        const data =await response.json();
        const challenges = data.challenges;

        const container = document.getElementById("challenge-container");

        container.innerHTML = "";

        challenges.forEach(challenge => {
            const card = createChallengesCard(challenge);
            container.appendChild(card);
        });
    }

    catch (error) {
        console.error("Error loading challenges.", error);
    }
}


//This will create the card for the challenges.

function createChallengesCard(challenge) {
    const div = document.createElement("div");
    div.className = "challenge-card";

    div.innerHTML = `
    <h3>${challenge.name}</h3>
    <p>${challenge.description}</p>
    <span class="tag">${challenge.category}</span>
    <button onclick= "startChallenge('${challenge.name}')">
        Start Challenge
    </button>
    
    
     `;

    return div;
}


//placeholder function it will be replaced for when we actually need to call backend
function startChallenge(name) {
    console.log("Starting challenge:", name);
    alert(`Starting ${name}`);
}
