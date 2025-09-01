<template>
  <div class="app">
    <!-- Header with gradient background -->
    <header class="header">
      <div class="header-content">
        <h1 class="title">🔥 Mythical Card Battle 🔥</h1>
        <p class="subtitle">Epic creatures clash in an ancient arena</p>
      </div>
    </header>

    <!-- Connection form with glass morphism effect -->
    <div v-if="!connected" class="connection-container">
      <div class="connection-card">
        <h2 class="connection-title">🎮 Join the Battle</h2>
        <div class="form-group">
          <label class="form-label">🏰 Room ID</label>
          <input 
            v-model="roomId" 
            placeholder="Enter room ID" 
            class="form-input"
          />
        </div>
        <div class="form-group">
          <label class="form-label">⚔️ Warrior Name</label>
          <input 
            v-model="playerName" 
            placeholder="Enter your name" 
            class="form-input"
          />
        </div>
        <button 
          @click="joinGame" 
          :disabled="!roomId || !playerName"
          class="join-button"
        >
          🚀 Enter Arena
        </button>
      </div>
    </div>

    <!-- Waiting message with animation -->
    <div v-if="waitingMessage" class="waiting-container">
      <div class="waiting-card">
        <div class="loading-spinner"></div>
        <h3 class="waiting-title">{{ waitingMessage }}</h3>
        <p class="waiting-subtitle">Summoning another warrior...</p>
      </div>
    </div>

    <!-- Game UI with epic styling -->
    <div v-if="myTopCard && opponentTopCard" class="game-container">
      <div class="game-header">
        <h2 class="game-title">⚔️ Battle in Progress ⚔️</h2>
        <div class="round-info">Round {{ currentRound || 1 }}</div>
      </div>

      <div class="battle-arena">
        <!-- Your card section -->
        <div class="player-section">
          <div class="player-header">
            <h3 class="player-title">🎯 Your Champion</h3>
            <div class="deck-size">Deck: {{ myDeckSize }} cards</div>
          </div>
          
          <div class="card-container your-card">
            <div class="card-image">
              <img 
                :src="myTopCard.image" 
                :alt="myTopCard.name" 
                @error="handleImageError"
                @load="handleImageLoad"
                style="border: 2px solid #ffd700;"
              />
              <div v-if="imageError" class="image-error">
                ❌ Image failed to load
              </div>
            </div>
            <h4 class="card-name">{{ myTopCard.name }}</h4>
            
            <div class="attributes">
              <h5>Choose Your Attack:</h5>
              <div class="attribute-buttons">
                <button 
                  v-for="(value, key) in myTopCard.stats" 
                  :key="key"
                  :disabled="!yourTurn"
                  @click="playAttribute(key)"
                  class="attribute-button"
                  :class="{ 'your-turn': yourTurn, 'disabled': !yourTurn }"
                >
                  <span class="attribute-name">{{ key }}</span>
                  <span class="attribute-value">{{ value }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- VS indicator -->
        <div class="vs-section">
          <div class="vs-circle">
            <span class="vs-text">VS</span>
          </div>
          <div class="turn-indicator" v-if="yourTurn">
            <span class="turn-text">🎯 Your Turn!</span>
          </div>
        </div>

        <!-- Opponent card section -->
        <div class="player-section">
          <div class="player-header">
            <h3 class="player-title">👹 Enemy Champion</h3>
            <div class="deck-size">Deck: {{ opponentDeckSize }} cards</div>
          </div>
          
          <div class="card-container opponent-card">
            <div class="card-image">
              <img 
                :src="opponentTopCard.image" 
                :alt="opponentTopCard.name" 
                @error="handleImageError"
                @load="handleImageLoad"
                style="border: 2px solid #ff6b6b;"
              />
              <div v-if="imageError" class="image-error">
                ❌ Image failed to load
              </div>
            </div>
            <h4 class="card-name">{{ opponentTopCard.name }}</h4>
            <div class="opponent-stats">
              <div v-for="(value, key) in opponentTopCard.stats" :key="key" class="stat-item">
                <span class="stat-name">{{ key }}:</span>
                <span class="stat-value">{{ value }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Game status and results -->
      <div class="game-status">
        <div v-if="winner" class="result-banner winner">
          🏆 Round Winner: {{ winner }} 🏆
        </div>
        <div v-if="gameOver" class="result-banner game-over">
          🎉 Game Over! Winner: {{ gameOver }} 🎉
        </div>
      </div>
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
const currentRound = ref(1)
const imageError = ref(false)

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
        currentRound.value = 1
        imageError.value = false
        
        // Debug: Log the image URLs
        console.log("Your card image URL:", myTopCard.value?.image)
        console.log("Opponent card image URL:", opponentTopCard.value?.image)
        break

      case "roundResult":
        myTopCard.value = msg.yourTopCard
        opponentTopCard.value = msg.opponentTopCard
        myDeckSize.value = msg.yourDeckSize
        opponentDeckSize.value = msg.opponentDeckSize
        winner.value = msg.winner
        gameOver.value = msg.gameOver
        yourTurn.value = msg.yourTurn
        if (winner.value && !gameOver.value) {
          currentRound.value++
        }
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

// Handle image loading errors
function handleImageError(event) {
  console.error("Image failed to load:", event.target.src)
  
  // Try fallback image
  if (!event.target.src.includes('fallback')) {
    event.target.src = 'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMTUwIiBoZWlnaHQ9IjE1MCIgdmlld0JveD0iMCAwIDE1MCAxNTAiIGZpbGw9Im5vbmUiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PHJlY3Qgd2lkdGg9IjE1MCIgaGVpZ2h0PSIxNTAiIGZpbGw9IiM2NjYiIHJ4PSIxMCIvPjx0ZXh0IHg9Ijc1IiB5PSI3NSIgZm9udC1mYW1pbHk9IkFyaWFsIiBmb250LXNpemU9IjE0IiBmaWxsPSJ3aGl0ZSIgdGV4dC1hbmNob3I9Im1pZGRsZSI+Q2FyZCBJbWFnZTwvdGV4dD48L3N2Zz4='
  } else {
    imageError.value = true
  }
}

// Handle successful image loading
function handleImageLoad(event) {
  console.log("Image loaded successfully:", event.target.src)
  imageError.value = false
}
</script>

<style scoped>
/* Global app styles */
.app {
  min-height: 100vh;
  background: linear-gradient(135deg, #0f0c29 0%, #302b63 50%, #24243e 100%);
  font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
  color: white;
}

/* Header styles */
.header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 2rem 0;
  text-align: center;
  box-shadow: 0 4px 20px rgba(0,0,0,0.3);
}

.header-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 2rem;
}

.title {
  font-size: 3.5rem;
  font-weight: 800;
  margin: 0;
  background: linear-gradient(45deg, #ffd700, #ffed4e);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  text-shadow: 2px 2px 4px rgba(0,0,0,0.3);
}

.subtitle {
  font-size: 1.2rem;
  margin: 0.5rem 0 0 0;
  opacity: 0.9;
  font-weight: 300;
}

/* Connection form styles */
.connection-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 60vh;
  padding: 2rem;
}

.connection-card {
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(20px);
  border-radius: 20px;
  padding: 3rem;
  box-shadow: 0 8px 32px rgba(0,0,0,0.3);
  border: 1px solid rgba(255, 255, 255, 0.2);
  max-width: 500px;
  width: 100%;
}

.connection-title {
  text-align: center;
  font-size: 2rem;
  margin-bottom: 2rem;
  color: #ffd700;
}

.form-group {
  margin-bottom: 1.5rem;
}

.form-label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 600;
  color: #e0e0e0;
}

.form-input {
  width: 100%;
  padding: 1rem;
  border: none;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.1);
  color: white;
  font-size: 1rem;
  border: 1px solid rgba(255, 255, 255, 0.2);
  transition: all 0.3s ease;
}

.form-input:focus {
  outline: none;
  border-color: #ffd700;
  box-shadow: 0 0 20px rgba(255, 215, 0, 0.3);
  background: rgba(255, 255, 255, 0.15);
}

.form-input::placeholder {
  color: rgba(255, 255, 255, 0.6);
}

.join-button {
  width: 100%;
  padding: 1rem 2rem;
  background: linear-gradient(45deg, #ff6b6b, #ee5a24);
  border: none;
  border-radius: 10px;
  color: white;
  font-size: 1.2rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 4px 15px rgba(255, 107, 107, 0.4);
}

.join-button:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(255, 107, 107, 0.6);
}

.join-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none;
}

/* Waiting styles */
.waiting-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 60vh;
  padding: 2rem;
}

.waiting-card {
  text-align: center;
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(20px);
  border-radius: 20px;
  padding: 3rem;
  box-shadow: 0 8px 32px rgba(0,0,0,0.3);
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.loading-spinner {
  width: 60px;
  height: 60px;
  border: 4px solid rgba(255, 255, 255, 0.3);
  border-top: 4px solid #ffd700;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto 2rem;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.waiting-title {
  font-size: 1.5rem;
  margin-bottom: 1rem;
  color: #ffd700;
}

.waiting-subtitle {
  opacity: 0.8;
  font-size: 1.1rem;
}

/* Game styles */
.game-container {
  max-width: 1400px;
  margin: 0 auto;
  padding: 2rem;
}

.game-header {
  text-align: center;
  margin-bottom: 3rem;
}

.game-title {
  font-size: 2.5rem;
  margin-bottom: 1rem;
  color: #ffd700;
  text-shadow: 2px 2px 4px rgba(0,0,0,0.5);
}

.round-info {
  font-size: 1.2rem;
  opacity: 0.8;
  background: rgba(255, 255, 255, 0.1);
  padding: 0.5rem 1.5rem;
  border-radius: 20px;
  display: inline-block;
}

.battle-arena {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  gap: 3rem;
  align-items: center;
  margin-bottom: 3rem;
}

.player-section {
  text-align: center;
}

.player-header {
  margin-bottom: 2rem;
}

.player-title {
  font-size: 1.8rem;
  margin-bottom: 0.5rem;
  color: #ffd700;
}

.deck-size {
  font-size: 1rem;
  opacity: 0.8;
  background: rgba(255, 255, 255, 0.1);
  padding: 0.3rem 1rem;
  border-radius: 15px;
  display: inline-block;
}

.card-container {
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(20px);
  border-radius: 20px;
  padding: 2rem;
  box-shadow: 0 8px 32px rgba(0,0,0,0.3);
  border: 1px solid rgba(255, 255, 255, 0.2);
  transition: all 0.3s ease;
}

.card-container:hover {
  transform: translateY(-5px);
  box-shadow: 0 12px 40px rgba(0,0,0,0.4);
}

.card-image {
  margin-bottom: 1rem;
}

.card-image img {
  width: 150px;
  height: 150px;
  border-radius: 15px;
  box-shadow: 0 4px 15px rgba(0,0,0,0.3);
}

.image-error {
  color: #ff6b6b;
  font-size: 0.9rem;
  margin-top: 0.5rem;
  padding: 0.5rem;
  background: rgba(255, 107, 107, 0.1);
  border-radius: 8px;
  border: 1px solid rgba(255, 107, 107, 0.3);
}

.card-name {
  font-size: 1.5rem;
  margin-bottom: 1.5rem;
  color: #ffd700;
}

.attributes h5 {
  margin-bottom: 1rem;
  color: #e0e0e0;
}

.attribute-buttons {
  display: flex;
  flex-direction: column;
  gap: 0.8rem;
}

.attribute-button {
  padding: 1rem;
  background: linear-gradient(45deg, #4ecdc4, #44a08d);
  border: none;
  border-radius: 10px;
  color: white;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: 0 4px 15px rgba(78, 205, 196, 0.4);
}

.attribute-button:hover:not(.disabled) {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(78, 205, 196, 0.6);
}

.attribute-button.your-turn {
  background: linear-gradient(45deg, #ffd700, #ffed4e);
  color: #333;
  box-shadow: 0 4px 15px rgba(255, 215, 0, 0.4);
}

.attribute-button.your-turn:hover {
  box-shadow: 0 6px 20px rgba(255, 215, 0, 0.6);
}

.attribute-button.disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none;
}

.attribute-name {
  font-weight: 600;
}

.attribute-value {
  background: rgba(0, 0, 0, 0.2);
  padding: 0.3rem 0.8rem;
  border-radius: 8px;
  font-weight: 700;
}

.opponent-stats {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.stat-item {
  display: flex;
  justify-content: space-between;
  padding: 0.5rem;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 8px;
}

.stat-name {
  font-weight: 600;
  color: #e0e0e0;
}

.stat-value {
  font-weight: 700;
  color: #ffd700;
}

/* VS section */
.vs-section {
  text-align: center;
}

.vs-circle {
  width: 100px;
  height: 100px;
  background: linear-gradient(45deg, #ff6b6b, #ee5a24);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 1rem;
  box-shadow: 0 4px 20px rgba(255, 107, 107, 0.4);
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.05); }
}

.vs-text {
  font-size: 1.5rem;
  font-weight: 800;
  color: white;
}

.turn-indicator {
  background: linear-gradient(45deg, #ffd700, #ffed4e);
  color: #333;
  padding: 0.5rem 1rem;
  border-radius: 20px;
  font-weight: 600;
  animation: bounce 1s infinite;
}

@keyframes bounce {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-5px); }
}

/* Game status */
.game-status {
  text-align: center;
  margin-top: 2rem;
}

.result-banner {
  padding: 1rem 2rem;
  border-radius: 15px;
  font-size: 1.3rem;
  font-weight: 600;
  margin-bottom: 1rem;
  animation: slideIn 0.5s ease;
}

@keyframes slideIn {
  from { transform: translateY(-20px); opacity: 0; }
  to { transform: translateY(0); opacity: 1; }
}

.result-banner.winner {
  background: linear-gradient(45deg, #4ecdc4, #44a08d);
  color: white;
}

.result-banner.game-over {
  background: linear-gradient(45deg, #ffd700, #ffed4e);
  color: #333;
}

/* Responsive design */
@media (max-width: 768px) {
  .battle-arena {
    grid-template-columns: 1fr;
    gap: 2rem;
  }
  
  .title {
    font-size: 2.5rem;
  }
  
  .connection-card {
    padding: 2rem;
    margin: 1rem;
  }
  
  .game-container {
    padding: 1rem;
  }
}
</style>
