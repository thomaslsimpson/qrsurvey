(function () {
  'use strict';

  var dataEl = document.getElementById('survey-data');
  var data = JSON.parse(dataEl.textContent);
  var items = data.items || [];
  var N = items.length;

  var answers = {}; // surveyItemId -> value 1-5
  var step = 0; // 0=welcome, 1..N=questions, N+1=contest, N+2=thanks

  var els = {
    header: document.getElementById('wizard-header'),
    backBtn: document.getElementById('back-btn'),
    progressFill: document.getElementById('progress-fill'),
    countdown: document.getElementById('countdown'),
    screens: {
      welcome: document.getElementById('screen-welcome'),
      question: document.getElementById('screen-question'),
      contest: document.getElementById('screen-contest'),
      thanks: document.getElementById('screen-thanks')
    },
    startBtn: document.getElementById('start-btn'),
    qKicker: document.getElementById('q-kicker'),
    qPrompt: document.getElementById('q-prompt'),
    qCurrentLabel: document.getElementById('q-current-label'),
    qStarRow: document.getElementById('q-star-row'),
    qSliderTrack: document.getElementById('q-slider-track'),
    qSliderFill: document.getElementById('q-slider-fill'),
    qSliderDots: document.getElementById('q-slider-dots'),
    qSliderThumb: document.getElementById('q-slider-thumb'),
    qWorst: document.getElementById('q-worst'),
    qBest: document.getElementById('q-best'),
    nextBtn: document.getElementById('next-btn'),
    fName: document.getElementById('f-name'),
    fEmail: document.getElementById('f-email'),
    fPhone: document.getElementById('f-phone'),
    fHp: document.getElementById('f-hp'),
    submitError: document.getElementById('submit-error'),
    enterBtn: document.getElementById('enter-btn')
  };

  function screenForStep(i) {
    if (i === 0) return 'welcome';
    if (i >= 1 && i <= N) return 'question';
    if (i === N + 1) return 'contest';
    return 'thanks';
  }

  function currentItem() {
    return items[step - 1];
  }

  function reduceMotion() {
    return window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  }

  function render() {
    var screen = screenForStep(step);
    for (var key in els.screens) {
      if (Object.prototype.hasOwnProperty.call(els.screens, key)) {
        els.screens[key].hidden = key !== screen;
      }
    }

    var showHeader = step >= 1 && step <= N + 1;
    els.header.hidden = !showHeader;
    var progressPct = Math.min(100, Math.round((step / (N + 1)) * 100));
    els.progressFill.style.width = progressPct + '%';

    els.countdown.hidden = screen !== 'question';
    if (screen === 'question') {
      var remaining = N - (step - 1);
      els.countdown.textContent = (remaining === 1 ? '1 question' : remaining + ' questions') + ' until your entry';
      renderQuestion();
    }

    playEnterAnimation(screen);
  }

  function playEnterAnimation(screen) {
    var active = els.screens[screen];
    active.classList.remove('stage-anim', 'enter');
    if (reduceMotion()) return;
    void active.offsetWidth; // force reflow so the class re-triggers
    active.classList.add('stage-anim', 'enter');
    requestAnimationFrame(function () {
      requestAnimationFrame(function () {
        active.classList.remove('enter');
      });
    });
  }

  function renderQuestion() {
    var item = currentItem();
    var sel = answers[item.id];

    els.qKicker.textContent = 'Question ' + step + ' of ' + N;
    els.qPrompt.textContent = item.question;
    els.qWorst.textContent = item.responses[0];
    els.qBest.textContent = item.responses[4];

    if (sel) {
      els.qCurrentLabel.textContent = item.responses[sel - 1];
      els.qCurrentLabel.classList.remove('placeholder');
    } else {
      els.qCurrentLabel.textContent = 'Slide to rate →';
      els.qCurrentLabel.classList.add('placeholder');
    }

    els.qStarRow.innerHTML = '';
    var rank;
    for (rank = 1; rank <= 5; rank++) {
      (function (rank) {
        var col = document.createElement('div');
        col.className = 'star-col';
        for (var s = 0; s < rank; s++) {
          var star = document.createElement('span');
          star.className = 'star' + (sel === rank ? ' on' : '');
          star.textContent = '★';
          col.appendChild(star);
        }
        col.addEventListener('click', function () { selectValue(rank); });
        els.qStarRow.appendChild(col);
      })(rank);
    }

    els.qSliderDots.innerHTML = '';
    for (rank = 1; rank <= 5; rank++) {
      (function (rank) {
        var dotCol = document.createElement('div');
        dotCol.className = 'slider-dot-col';
        var dot = document.createElement('div');
        dot.className = 'slider-dot' + (sel === rank ? ' on' : '');
        dotCol.appendChild(dot);
        dotCol.addEventListener('click', function () { selectValue(rank); });
        els.qSliderDots.appendChild(dotCol);
      })(rank);
    }

    if (sel) {
      var center = (sel - 1) * 20 + 10; // percent along the 10%-90% track
      els.qSliderFill.style.width = (center - 10) + '%';
      els.qSliderThumb.style.left = center + '%';
      els.qSliderThumb.hidden = false;
    } else {
      els.qSliderFill.style.width = '0%';
      els.qSliderThumb.hidden = true;
    }

    els.nextBtn.disabled = !sel;
  }

  function selectValue(rank) {
    answers[currentItem().id] = rank;
    renderQuestion();
  }

  function trackPick(e) {
    var rect = els.qSliderTrack.getBoundingClientRect();
    var frac = (e.clientX - rect.left) / rect.width;
    if (frac < 0) frac = 0;
    if (frac > 1) frac = 1;
    var rank = Math.min(4, Math.max(0, Math.round(frac * 5 - 0.5))) + 1;
    selectValue(rank);
  }

  var pressed = false;
  els.qSliderTrack.addEventListener('pointerdown', function (e) {
    pressed = true;
    try { els.qSliderTrack.setPointerCapture(e.pointerId); } catch (err) { /* ignore */ }
    trackPick(e);
  });
  els.qSliderTrack.addEventListener('pointermove', function (e) { if (pressed) trackPick(e); });
  els.qSliderTrack.addEventListener('pointerup', function () { pressed = false; });

  function goTo(i) {
    step = Math.max(0, Math.min(N + 2, i));
    render();
  }

  els.startBtn.addEventListener('click', function () { goTo(1); });
  els.backBtn.addEventListener('click', function () { goTo(step - 1); });
  els.nextBtn.addEventListener('click', function () {
    if (answers[currentItem().id]) goTo(step + 1);
  });

  function showError(msg) {
    els.submitError.textContent = msg;
    els.submitError.hidden = false;
  }

  els.enterBtn.addEventListener('click', function () {
    els.submitError.hidden = true;
    var name = els.fName.value.trim();
    var phone = els.fPhone.value.trim();
    if (!name || !phone) {
      showError('Please fill in your name and mobile number.');
      return;
    }

    var payload = {
      name: name,
      email: els.fEmail.value.trim(),
      phone: phone,
      honeypot: els.fHp.value,
      answers: Object.keys(answers).map(function (id) {
        return { survey_item_id: Number(id), value_selected: answers[id] };
      })
    };

    els.enterBtn.disabled = true;
    fetch(window.location.pathname.replace(/\/$/, '') + '/submit', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    }).then(function (res) {
      if (res.ok) {
        goTo(N + 2);
        return;
      }
      return res.json().catch(function () { return {}; }).then(function (body) {
        showError((body && body.error) || 'Something went wrong — please try again.');
      });
    }).catch(function () {
      showError('Network error — please check your connection and try again.');
    }).finally(function () {
      els.enterBtn.disabled = false;
    });
  });

  render();
})();
