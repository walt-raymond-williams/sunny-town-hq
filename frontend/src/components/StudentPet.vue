<script setup lang="ts">
defineProps({
  mood: {
    type: String,
    default: 'idle',
  },
})
</script>

<template>
  <aside class="student-pet" aria-label="Student pet" :data-mood="mood" role="img">
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
      <span class="student-pet__cheer student-pet__cheer--left" />
      <span class="student-pet__cheer student-pet__cheer--right" />
      <span class="student-pet__snack" />
      <span class="student-pet__tear student-pet__tear--left" />
      <span class="student-pet__tear student-pet__tear--right" />
      <span class="student-pet__growl student-pet__growl--left" />
      <span class="student-pet__growl student-pet__growl--right" />
      <span class="student-pet__sleep student-pet__sleep--one">Z</span>
      <span class="student-pet__sleep student-pet__sleep--two">Z</span>
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

.student-pet[data-mood='happy'] {
  --pet-body: #8ae36f;
  --pet-body-dark: #49b868;
  --pet-belly: #fff2a8;
}

.student-pet[data-mood='eating'] {
  --pet-body: #7ad7b7;
  --pet-body-dark: #3fa488;
  --pet-belly: #ffe0a6;
}

.student-pet[data-mood='hungry'] {
  --pet-body: #e2c66f;
  --pet-body-dark: #b89638;
  --pet-belly: #fff4c1;
}

.student-pet[data-mood='sad'] {
  --pet-body: #8bc3df;
  --pet-body-dark: #4e8fb0;
  --pet-belly: #d9f2ff;
}

.student-pet[data-mood='sleeping'] {
  --pet-body: #a8a9d9;
  --pet-body-dark: #7175b6;
  --pet-belly: #eeebff;
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

.student-pet[data-mood='happy'] .student-pet__body {
  animation: pet-happy-hop 0.72s ease-in-out infinite;
}

.student-pet[data-mood='eating'] .student-pet__body {
  animation: pet-munch-bob 0.52s ease-in-out infinite;
}

.student-pet[data-mood='hungry'] .student-pet__body {
  animation: pet-hungry-wobble 1s ease-in-out infinite;
}

.student-pet[data-mood='sleeping'] .student-pet__body {
  animation: pet-sleep-breathe 2.4s ease-in-out infinite;
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

.student-pet[data-mood='happy'] .student-pet__eye {
  animation: none;
  height: 20%;
  top: 18%;
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

.student-pet[data-mood='happy'] .student-pet__smile {
  bottom: 1%;
  height: 40%;
  left: 25%;
  width: 50%;
}

.student-pet[data-mood='eating'] .student-pet__smile {
  animation: pet-mouth-munch 0.52s ease-in-out infinite;
  border-radius: 50%;
  bottom: 4%;
  height: 24%;
  left: 37%;
  width: 26%;
}

.student-pet[data-mood='sad'] .student-pet__smile {
  border-bottom: 0;
  border-top: 3px solid var(--pet-face);
  border-radius: 999px 999px 0 0;
  bottom: -4%;
}

.student-pet[data-mood='sleeping'] .student-pet__eye {
  animation: none;
  height: 4%;
  top: 24%;
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

.student-pet[data-mood='happy'] .student-pet__sparkle {
  animation: pet-sparkle-spin 0.9s linear infinite;
}

.student-pet__cheer {
  background: #f0b22b;
  border-radius: 999px;
  height: 18%;
  opacity: 0;
  position: absolute;
  top: -2%;
  width: 5%;
}

.student-pet__cheer--left {
  left: -4%;
  transform: rotate(-28deg);
}

.student-pet__cheer--right {
  right: -4%;
  transform: rotate(28deg);
}

.student-pet[data-mood='happy'] .student-pet__cheer {
  animation: pet-cheer-pop 0.72s ease-in-out infinite;
  opacity: 1;
}

.student-pet__snack {
  background:
    radial-gradient(circle at 34% 34%, #7b4b24 0 8%, transparent 9%),
    radial-gradient(circle at 66% 42%, #7b4b24 0 8%, transparent 9%),
    radial-gradient(circle at 48% 68%, #7b4b24 0 7%, transparent 8%),
    #d99a45;
  border: 2px solid #7b4b24;
  border-radius: 50%;
  height: 22%;
  opacity: 0;
  position: absolute;
  right: -10%;
  top: 42%;
  width: 22%;
}

.student-pet[data-mood='eating'] .student-pet__snack {
  animation: pet-cookie-bite 1.3s ease-in-out infinite;
  opacity: 1;
}

.student-pet__tear {
  background: #d9f2ff;
  border: 2px solid #27746f;
  border-radius: 70% 70% 70% 10%;
  height: 15%;
  opacity: 0;
  position: absolute;
  top: 42%;
  width: 8%;
}

.student-pet__tear--left {
  left: 28%;
  transform: rotate(-18deg);
}

.student-pet__tear--right {
  right: 28%;
  transform: rotate(18deg);
}

.student-pet[data-mood='sad'] .student-pet__tear {
  animation: pet-tear-drop 1.4s ease-in-out infinite;
  opacity: 1;
}

.student-pet__growl {
  border: 2px solid #7b4b24;
  border-left: 0;
  border-right: 0;
  border-top: 0;
  border-radius: 0 0 999px 999px;
  bottom: 22%;
  height: 9%;
  opacity: 0;
  position: absolute;
  width: 18%;
}

.student-pet__growl--left {
  left: 13%;
}

.student-pet__growl--right {
  right: 13%;
}

.student-pet[data-mood='hungry'] .student-pet__growl {
  animation: pet-tummy-growl 0.8s ease-in-out infinite;
  opacity: 1;
}

.student-pet__sleep {
  color: #17212b;
  font-size: 18px;
  font-weight: 900;
  opacity: 0;
  position: absolute;
  right: -12%;
}

.student-pet__sleep--one {
  top: -8%;
}

.student-pet__sleep--two {
  top: -24%;
}

.student-pet[data-mood='sleeping'] .student-pet__sleep {
  animation: pet-sleep-float 1.9s ease-in-out infinite;
  opacity: 1;
}

.student-pet[data-mood='sleeping'] .student-pet__sleep--two {
  animation-delay: 0.5s;
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

.student-pet[data-mood='happy'] .student-pet__shadow {
  animation: pet-happy-shadow 0.72s ease-in-out infinite;
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

@keyframes pet-happy-hop {
  0%,
  100% {
    transform: translateY(0) scaleX(1.04) scaleY(0.96) rotate(-3deg);
  }

  50% {
    transform: translateY(-17px) scaleX(0.96) scaleY(1.05) rotate(3deg);
  }
}

@keyframes pet-munch-bob {
  0%,
  100% {
    transform: translateY(0) scaleX(1.02) rotate(-1deg);
  }

  50% {
    transform: translateY(-5px) scaleX(0.98) rotate(1deg);
  }
}

@keyframes pet-hungry-wobble {
  0%,
  100% {
    transform: translateY(0) rotate(-2deg);
  }

  50% {
    transform: translateY(-3px) rotate(2deg);
  }
}

@keyframes pet-sleep-breathe {
  0%,
  100% {
    transform: translateY(0) scaleY(0.97);
  }

  50% {
    transform: translateY(-2px) scaleY(1.02);
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

@keyframes pet-happy-shadow {
  0%,
  100% {
    opacity: 0.5;
    transform: scaleX(1.08);
  }

  50% {
    opacity: 0.24;
    transform: scaleX(0.68);
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

@keyframes pet-mouth-munch {
  0%,
  100% {
    transform: scaleY(0.7);
  }

  50% {
    transform: scaleY(1.15);
  }
}

@keyframes pet-sparkle-spin {
  100% {
    transform: rotate(360deg);
  }
}

@keyframes pet-cookie-bite {
  0% {
    opacity: 0;
    transform: translate(12px, 4px) scale(0.82);
  }

  20% {
    opacity: 1;
  }

  72% {
    opacity: 1;
    transform: translate(-38px, -16px) scale(0.72);
  }

  100% {
    opacity: 0;
    transform: translate(-42px, -18px) scale(0.18);
  }
}

@keyframes pet-cheer-pop {
  0%,
  100% {
    transform: translateY(3px) scaleY(0.75);
  }

  50% {
    transform: translateY(-8px) scaleY(1.15);
  }
}

@keyframes pet-tear-drop {
  0%,
  100% {
    transform: translateY(0) scale(0.7);
  }

  50% {
    transform: translateY(10px) scale(1);
  }
}

@keyframes pet-tummy-growl {
  0%,
  100% {
    transform: scaleX(0.7);
  }

  50% {
    transform: scaleX(1.15);
  }
}

@keyframes pet-sleep-float {
  0% {
    opacity: 0;
    transform: translate(0, 8px) scale(0.8);
  }

  35% {
    opacity: 1;
  }

  100% {
    opacity: 0;
    transform: translate(8px, -10px) scale(1.1);
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
  .student-pet__cheer,
  .student-pet__eye,
  .student-pet__growl,
  .student-pet__snack,
  .student-pet__sleep,
  .student-pet__sparkle,
  .student-pet__tear,
  .student-pet__shadow {
    animation: none;
  }
}
</style>
