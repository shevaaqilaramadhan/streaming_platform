<template>
  <canvas ref="canvasRef" class="constellation-canvas"></canvas>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

const canvasRef = ref(null)

let ctx = null
let animationFrameId = null
let width = 0
let height = 0
let particles = []

// Mouse tracking state
const mouse = {
  x: null,
  y: null,
  radius: 150 // distance to connect to particles
}

// Particle Config
const config = {
  particleColor: 'rgba(255, 46, 147, 0.7)', // Faint electric pink
  lineColor: 'rgba(255, 0, 127, 0.15)', // Electric pink lines
  connectionDistance: 110,
  maxParticles: 75
}

class Particle {
  constructor() {
    this.x = Math.random() * width
    this.y = Math.random() * height
    this.vx = (Math.random() - 0.5) * 0.4 // Slow speeds for elegant feel
    this.vy = (Math.random() - 0.5) * 0.4
    this.radius = Math.random() * 1.5 + 1 // 1px to 2.5px
    this.alpha = Math.random() * 0.5 + 0.3
  }

  update() {
    this.x += this.vx
    this.y += this.vy

    // Bounce off edges
    if (this.x < 0 || this.x > width) this.vx = -this.vx
    if (this.y < 0 || this.y > height) this.vy = -this.vy
    
    // Clamp to screen bounds to prevent getting stuck
    if (this.x < 0) this.x = 0
    if (this.x > width) this.x = width
    if (this.y < 0) this.y = 0
    if (this.y > height) this.y = height
  }

  draw() {
    ctx.beginPath()
    ctx.arc(this.x, this.y, this.radius, 0, Math.PI * 2)
    ctx.fillStyle = `rgba(255, 46, 147, ${this.alpha})`
    ctx.fill()
  }
}

function handleResize() {
  if (!canvasRef.value) return
  const canvas = canvasRef.value
  width = canvas.width = window.innerWidth
  height = canvas.height = window.innerHeight

  // Adjust particle count dynamically based on viewport area
  const count = Math.min(
    Math.floor((width * height) / 16000),
    config.maxParticles
  )
  
  // Reinitialize particles if count changed significantly
  if (particles.length < count) {
    while (particles.length < count) {
      particles.push(new Particle())
    }
  } else if (particles.length > count) {
    particles.splice(count)
  }
}

function handleMouseMove(e) {
  mouse.x = e.clientX
  mouse.y = e.clientY
}

function handleMouseLeave() {
  mouse.x = null
  mouse.y = null
}

function animate() {
  ctx.clearRect(0, 0, width, height)

  // Update and draw particles
  for (let i = 0; i < particles.length; i++) {
    const p = particles[i]
    p.update()
    p.draw()
  }

  // Draw lines (constellations)
  for (let i = 0; i < particles.length; i++) {
    const p1 = particles[i]

    // Connection to other particles
    for (let j = i + 1; j < particles.length; j++) {
      const p2 = particles[j]
      const dx = p1.x - p2.x
      const dy = p1.y - p2.y
      const dist = Math.sqrt(dx * dx + dy * dy)

      if (dist < config.connectionDistance) {
        const alpha = (1 - dist / config.connectionDistance) * 0.15
        ctx.beginPath()
        ctx.moveTo(p1.x, p1.y)
        ctx.lineTo(p2.x, p2.y)
        ctx.strokeStyle = `rgba(255, 0, 127, ${alpha})`
        ctx.lineWidth = 0.8
        ctx.stroke()
      }
    }

    // Connection to mouse
    if (mouse.x !== null && mouse.y !== null) {
      const dx = p1.x - mouse.x
      const dy = p1.y - mouse.y
      const dist = Math.sqrt(dx * dx + dy * dy)

      if (dist < mouse.radius) {
        const alpha = (1 - dist / mouse.radius) * 0.25
        ctx.beginPath()
        ctx.moveTo(p1.x, p1.y)
        ctx.lineTo(mouse.x, mouse.y)
        ctx.strokeStyle = `rgba(255, 46, 147, ${alpha})`
        ctx.lineWidth = 1.0
        ctx.stroke()
      }
    }
  }

  animationFrameId = requestAnimationFrame(animate)
}

onMounted(() => {
  const canvas = canvasRef.value
  if (!canvas) return
  ctx = canvas.getContext('2d')
  
  handleResize()
  animate()

  window.addEventListener('resize', handleResize)
  window.addEventListener('mousemove', handleMouseMove)
  window.addEventListener('mouseleave', handleMouseLeave)
})

onUnmounted(() => {
  cancelAnimationFrame(animationFrameId)
  window.removeEventListener('resize', handleResize)
  window.removeEventListener('mousemove', handleMouseMove)
  window.removeEventListener('mouseleave', handleMouseLeave)
})
</script>

<style scoped>
.constellation-canvas {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  z-index: -1;
  pointer-events: none;
  background: transparent;
}
</style>
