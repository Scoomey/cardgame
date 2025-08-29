<template>
  <div>
    <h1>Mythical Card Game</h1>

    <!-- Connection form -->
    <div v-if="!connected">
      <div>
        <label>Room ID: </label>
        <input v-model="roomId" placeholder="Enter room ID" />
      </div>
      <div>
        <label>Player Name: </label>
        <input v-model="playerName" placeholder="Enter your name" />
      </div>
      <button @click="joinGame" :disabled="!roomId || !playerName">Join Game</button>
    </div>

    <!-- Waiting message -->
    <div v-if="waitingMessage">{{ waitingMessage }}</div>

    <!-- Game UI -->
    <div v-if="myTopCard && opponentTopCard">
      <div style="display:flex; gap: 50px;">
        <div>
          <h2>Your Top Card: {{ myTopCard.name }}</h2>
          <img :src="myTopCard.image" width="150" />
          <h3>Choose Attribute</h3>
          <ul>
            <li v-for="(value, key) in myTopCard.stats" :key="key">
              <button :disabled="!yourTurn" @click="playAttribute(key)">
                {{ key }}: {{ value }}
              </button>
            </li>
          </ul>
          <p>Your Deck: {{ myDeckSize }} cards</p>
        </div>

        <div>
          <h2>Opponent Top Card: {{ opponentTopCard.name }}</h2>
          <img :src="opponentTopCard.image" width="150" />
          <p>Opponent Deck: {{ opponentDeckSize }} cards</p>
        </div>
      </div>

      <p v-if="winner">Round Winner: {{ winner }}</p>
      <p v-if="gameOver">Game Over! Winner: {{ gameOver }}</p>
      <p v-if="yourTurn">It's your turn!</p>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

// Reactive state
const myTopCard = ref(null)
const opponentTopCard = ref(null)
const myDeckSize = ref(0)
const opponentDeckSize = ref(0)
const yourTurn = ref(false)
const winner = ref(null)
const gameOver = ref(null)
const waitingMessage = ref("")
const connected = ref(false)
const roomId = ref("")
const playerName = ref("")

let ws

onMounted(() => {
  // Generate a unique player ID
  const uniqueId = 'player_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9)
  playerName.value = uniqueId
})

function joinGame() {
  if (!roomId.value || !playerName.value) return
  
  // Connect to EC2 backend
  ws = new WebSocket("ws://ec2-13-223-180-228.compute-1.amazonaws.com:8080/ws")

  ws.onopen = () => {
    console.log("Connected to backend!")
    ws.send(JSON.stringify({
      action: "join",
      room: roomId.value,
      player: playerName.value
    }))
    connected.value = true
  }

  ws.onmessage = (event) => {
    const msg = JSON.parse(event.data)
    console.log("Received message:", msg)

    switch(msg.action){
      case "waiting":
        waitingMessage.value = msg.message
        break

      case "start":
        waitingMessage.value = ""
        myTopCard.value = msg.yourTopCard
        opponentTopCard.value = msg.opponentTopCard
        myDeckSize.value = msg.yourDeckSize
        opponentDeckSize.value = msg.opponentDeckSize
        yourTurn.value = msg.yourTurn
        winner.value = null
        gameOver.value = null
        break

      case "roundResult":
        myTopCard.value = msg.yourTopCard
        opponentTopCard.value = msg.opponentTopCard
        myDeckSize.value = msg.yourDeckSize
        opponentDeckSize.value = msg.opponentDeckSize
        winner.value = msg.winner
        gameOver.value = msg.gameOver
        yourTurn.value = msg.yourTurn
        break
    }
  }

  ws.onerror = (err) => {
    console.error("WebSocket error:", err)
  }
}

// Send chosen attribute to backend
function playAttribute(attr){
  if (!myTopCard.value) return
  ws.send(JSON.stringify({
    action: "playCard",
    card: { name: myTopCard.value.name },
    attribute: attr
  }))
  yourTurn.value = false
}
</script>

<style scoped>
button {
  margin: 5px;
  padding: 5px 10px;
  cursor: pointer;
}
button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
