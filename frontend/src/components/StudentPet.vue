<script setup>
defineProps({
  mood: {
    type: String,
    default: 'idle',
  },
})
</script>

<template>
  <aside class="student-pet" :data-mood="mood" aria-hidden="true">
    <div class="student-pet__shadow" />
    <div class="student-pet__body">
      <div class="student-pet__ear student-pet__ear--left" />
      <div class="student-pet__ear student-pet__ear--right" />
      <div class="student-pet__face">
        <span class="student-pet__eye student-pet__eye--left" />
        <span class="student-pet__eye student-pet__eye--right" />
        <span class="student-pet__smile" />
      </div>
      <div class="student-pet__belly">
        <span class="student-pet__sparkle" />
      </div>
    </div>
  </aside>
</template>

<style scoped>
.student-pet {
  --pet-body: #6bcf9f;
  --pet-body-dark: #48a97d;
  --pet-face: #17212b;
  --pet-belly: #f9f2cf;
  bottom: 18px;
  height: clamp(104px, 14vw, 148px);
  pointer-events: none;
  position: fixed;
  right: 18px;
  width: clamp(104px, 14vw, 148px);
  z-index: 20;
}

.student-pet__body {
  animation: pet-idle-bob 2.8s ease-in-out infinite;
  background:
    radial-gradient(circle at 34% 28%, rgba(255, 255, 255, 0.52), transparent 16%),
    linear-gradient(145deg, var(--pet-body), var(--pet-body-dark));
  border: 3px solid #17212b;
  border-radius: 48% 52% 42% 46%;
  box-shadow: 0 10px 0 rgba(23, 33, 43, 0.08) inset;
  height: 76%;
  left: 11%;
  position: absolute;
  top: 6%;
  transform-origin: center bottom;
  width: 78%;
}

.student-pet__ear {
  background: var(--pet-body);
  border: 3px solid #17212b;
  border-bottom: 0;
  height: 28%;
  position: absolute;
  top: -13%;
  width: 25%;
  z-index: -1;
}

.student-pet__ear--left {
  border-radius: 80% 30% 40% 20%;
  left: 14%;
  transform: rotate(-16deg);
}

.student-pet__ear--right {
  border-radius: 30% 80% 20% 40%;
  right: 14%;
  transform: rotate(16deg);
}

.student-pet__face {
  height: 34%;
  left: 20%;
  position: absolute;
  top: 25%;
  width: 60%;
}

.student-pet__eye {
  animation: pet-blink 5.4s infinite;
  background: var(--pet-face);
  border-radius: 999px;
  height: 30%;
  position: absolute;
  top: 12%;
  width: 16%;
}

.student-pet__eye--left {
  left: 14%;
}

.student-pet__eye--right {
  right: 14%;
}

.student-pet__smile {
  border: 3px solid var(--pet-face);
  border-left: 0;
  border-right: 0;
  border-top: 0;
  border-radius: 0 0 999px 999px;
  bottom: 8%;
  height: 28%;
  left: 31%;
  position: absolute;
  width: 38%;
}

.student-pet__belly {
  align-items: center;
  background: var(--pet-belly);
  border: 3px solid rgba(23, 33, 43, 0.7);
  border-radius: 50%;
  bottom: 8%;
  display: flex;
  height: 29%;
  justify-content: center;
  left: 32%;
  position: absolute;
  width: 36%;
}

.student-pet__sparkle {
  background: #27746f;
  clip-path: polygon(50% 0, 62% 38%, 100% 50%, 62% 62%, 50% 100%, 38% 62%, 0 50%, 38% 38%);
  height: 38%;
  opacity: 0.78;
  width: 38%;
}

.student-pet__shadow {
  animation: pet-shadow-pulse 2.8s ease-in-out infinite;
  background: rgba(23, 33, 43, 0.2);
  border-radius: 50%;
  bottom: 0;
  filter: blur(1px);
  height: 14%;
  left: 19%;
  position: absolute;
  width: 62%;
}

@keyframes pet-idle-bob {
  0%,
  100% {
    transform: translateY(0) rotate(-1deg);
  }

  50% {
    transform: translateY(-9px) rotate(1deg);
  }
}

@keyframes pet-shadow-pulse {
  0%,
  100% {
    opacity: 0.55;
    transform: scaleX(1);
  }

  50% {
    opacity: 0.35;
    transform: scaleX(0.82);
  }
}

@keyframes pet-blink {
  0%,
  88%,
  94%,
  100% {
    transform: scaleY(1);
  }

  91% {
    transform: scaleY(0.12);
  }
}

@media (max-width: 640px) {
  .student-pet {
    bottom: 10px;
    height: 92px;
    right: 10px;
    width: 92px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .student-pet__body,
  .student-pet__eye,
  .student-pet__shadow {
    animation: none;
  }
}
</style>
