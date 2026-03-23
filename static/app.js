document.addEventListener('DOMContentLoaded', () => {
    const loadingState = document.getElementById('loading-state');
    const errorState = document.getElementById('error-state');
    const errorMessage = document.getElementById('error-message');
    const regionsContainer = document.getElementById('regions-container');
    const lastUpdated = document.getElementById('last-updated');

    // Fetch and display status
    fetch('/api/status')
        .then(res => {
            if (!res.ok) throw new Error("Failed to fetch status");
            return res.json();
        })
        .then(status => {
            if (!lastUpdated) return;
            const parts = [];
            if (status.last_updated) {
                parts.push(`Last Updated: ${new Date(status.last_updated).toLocaleString()}`);
            }
            if (status.data_through) {
                const d = new Date(status.data_through);
                parts.push(`Data through: ${d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}`);
            }
            if (parts.length > 0) {
                lastUpdated.textContent = parts.join('  ·  ');
                showState(lastUpdated);
            }
        })
        .catch(err => console.error("Status fetch error:", err));

    // Fetch Regions first
    fetch('/api/regions')
        .then(res => {
            if (!res.ok) throw new Error("Failed to fetch regions");
            return res.json();
        })
        .then(regions => {
            // Initiate parallel fetches for every single region
            const fetchPromises = regions.map(r =>
                Promise.all([
                    fetch(`/api/records?region=${r.id}`).then(res => {
                        if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
                        return res.json();
                    }),
                    fetch(`/api/recent?region=${r.id}`).then(res => {
                        if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
                        return res.json();
                    }),
                ]).then(([records, recent]) => ({ region: r, data: records, recent }))
            );

            return Promise.all(fetchPromises);
        })
        .then(results => {
            hideState(loadingState);
            showState(regionsContainer);

            // Render a card for each successfully fetched region
            results.forEach(result => {
                if (result.data && result.data.length > 0) {
                    renderRegionCard(result.region, result.data, result.recent);
                }
            });
        })
        .catch(err => {
            console.error("Fetch error:", err);
            showError("Failed to fetch dataset. The backend may still be caching historical records. Please refresh the page in a few seconds.");
        });

    // Builds the recent activity strip: one square per day for the last N days.
    // Colors: blue (below avg) → neutral (at avg) → dark red (above avg, non-record) → bright red (record).
    function buildRecentStrip(recentDays, currentYear) {
        if (!recentDays || recentDays.length === 0) return '';

        const n = recentDays.length;
        const vbW = n * 5;
        const vbH = 20;
        const gap = 0.6;
        const squareW = 5 - gap;
        const cornerR = 1;

        // Normalize deviations against the max absolute deviation in this window.
        const daysWithAvg = recentDays.filter(d => d.clim_avg > -900);
        const maxAbsDev = daysWithAvg.length > 0
            ? Math.max(...daysWithAvg.map(d => Math.abs(d.temp - d.clim_avg)))
            : 1;

        function squareColor(d) {
            // Record days: bright vivid red, clearly stronger than any non-record shade.
            if (d.is_record) return '#ef4444';
            // No climatological mean available: neutral gray.
            if (d.clim_avg < -900) return '#334155';

            const dev = d.temp - d.clim_avg;
            const t = Math.min(1, Math.abs(dev) / maxAbsDev);

            if (dev > 0) {
                // Neutral gray (#334155) → dark red (#991b1b) for above-average non-record days.
                // Records (#ef4444) are clearly brighter than the darkest red here.
                const r = Math.round(51 + (153 - 51) * t);
                const g = Math.round(65 + (27 - 65) * t);
                const b = Math.round(85 + (27 - 85) * t);
                return `rgb(${r},${g},${b})`;
            } else {
                // Neutral gray (#334155) → bright blue (#2563eb) for below-average days.
                const r = Math.round(51 + (37 - 51) * t);
                const g = Math.round(65 + (99 - 65) * t);
                const b = Math.round(85 + (235 - 85) * t);
                return `rgb(${r},${g},${b})`;
            }
        }

        function dayLabel(dayIndex) {
            const date = new Date(currentYear, 0, 1);
            date.setDate(date.getDate() + dayIndex);
            return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
        }

        let rects = '';
        recentDays.forEach((d, i) => {
            const x = (i * 5 + gap / 2).toFixed(2);
            const color = squareColor(d);
            const devStr = d.clim_avg > -900
                ? ` (${d.temp > d.clim_avg ? '+' : ''}${(d.temp - d.clim_avg).toFixed(2)}°C vs avg)`
                : '';
            const label = `${dayLabel(d.day_index)}${d.is_record ? ' · Record!' : ''} · ${d.temp.toFixed(2)}°C${devStr}`;
            rects += `<rect x="${x}" y="0" width="${squareW.toFixed(2)}" height="${vbH}" fill="${color}" rx="${cornerR}"><title>${label}</title></rect>`;
        });

        return `
            <div class="px-3 pt-3 pb-3 border-b border-slate-700 bg-slate-900/40">
                <div class="text-[10px] text-slate-500 uppercase tracking-wider mb-1.5 font-semibold flex justify-between">
                    <span>← ${n} days ago</span><span>today →</span>
                </div>
                <svg viewBox="0 0 ${vbW} ${vbH}" preserveAspectRatio="none" style="width:100%;height:20px;display:block;">${rects}</svg>
            </div>`;
    }

    // Builds a full-width SVG bar chart sparkline showing YTD records across all years.
    // Bars are heat-colored blue→red. The current year bar is highlighted white with a glow.
    function buildSparkline(data, currentYear, regionId) {
        const vbW = 600, vbH = 64;
        const padT = 5, padB = 8, padL = 2, padR = 2;
        const plotW = vbW - padL - padR;
        const plotH = vbH - padT - padB;

        const ytdVals = data.map(d => d.ytd_records);
        const minVal = Math.min(...ytdVals);
        const maxVal = Math.max(...ytdVals);
        const range = maxVal - minVal || 1;
        const mean = ytdVals.reduce((a, b) => a + b, 0) / ytdVals.length;

        const n = data.length;
        const barW = plotW / n;
        const gap = Math.max(0.3, barW * 0.12);

        // Interpolate between blue (#3b82f6) and red (#ef4444)
        function heatColor(t) {
            const r = Math.round(59 + (239 - 59) * t);
            const g = Math.round(130 + (68 - 130) * t);
            const b = Math.round(246 + (68 - 246) * t);
            return `rgb(${r},${g},${b})`;
        }

        const meanY = (padT + plotH - ((mean - minVal) / range) * plotH).toFixed(2);

        let rects = '';
        data.forEach((d, i) => {
            const isCurrent = d.year === currentYear;
            const t = (d.ytd_records - minVal) / range;
            const barH = Math.max(1, t * plotH);
            const x = (padL + i * barW + gap / 2).toFixed(2);
            const y = (padT + plotH - barH).toFixed(2);
            const w = (barW - gap).toFixed(2);
            const h = barH.toFixed(2);
            const title = `<title>${d.year}: ${d.ytd_records} record${d.ytd_records !== 1 ? 's' : ''}</title>`;

            if (isCurrent) {
                rects += `<rect x="${x}" y="${y}" width="${w}" height="${h}" fill="#f8fafc" rx="0.5" filter="url(#sparkglow-${regionId})">${title}</rect>`;
            } else {
                rects += `<rect x="${x}" y="${y}" width="${w}" height="${h}" fill="${heatColor(t)}" rx="0.5" opacity="0.85">${title}</rect>`;
            }
        });

        return `<svg viewBox="0 0 ${vbW} ${vbH}" preserveAspectRatio="none" style="width:100%;height:64px;display:block;">
            <defs>
                <filter id="sparkglow-${regionId}" x="-20%" y="-20%" width="140%" height="140%">
                    <feGaussianBlur stdDeviation="1.5" result="blur"/>
                    <feMerge><feMergeNode in="blur"/><feMergeNode in="SourceGraphic"/></feMerge>
                </filter>
            </defs>
            <line x1="${padL}" y1="${vbH - padB}" x2="${vbW - padR}" y2="${vbH - padB}" stroke="#334155" stroke-width="0.5"/>
            <line x1="${padL}" y1="${meanY}" x2="${vbW - padR}" y2="${meanY}" stroke="#475569" stroke-width="0.4" stroke-dasharray="3,3"/>
            ${rects}
        </svg>`;
    }

    // Builds a badge showing the current year's YTD vs the 5-year rolling average.
    function buildTrendBadge(data, currentYear) {
        const currentIdx = data.findIndex(d => d.year === currentYear);
        if (currentIdx < 5) return '';

        const priorFive = data.slice(currentIdx - 5, currentIdx);
        const avg = priorFive.reduce((sum, d) => sum + d.ytd_records, 0) / 5;
        if (avg === 0) return '';

        const current = data[currentIdx].ytd_records;
        const pct = Math.round(((current - avg) / avg) * 100);
        const isAbove = pct >= 0;
        const arrow = isAbove ? '↑' : '↓';
        const sign = isAbove ? '+' : '';
        const colorClass = isAbove ? 'text-red-400' : 'text-emerald-400';

        return `<div class="mt-2 text-sm font-semibold ${colorClass} tracking-wide">${arrow} ${sign}${pct}% vs 5yr avg</div>`;
    }

    function renderRegionCard(region, data, recentDays) {
        const currentYearData = data[data.length - 1];
        const sortedByYtd = [...data].sort((a, b) => b.ytd_records - a.ytd_records).slice(0, 5);
        const sortedRecent = [...data].sort((a, b) => b.year - a.year);
        const maxYtd = Math.max(...data.map(d => d.ytd_records));

        const card = document.createElement('div');
        card.className = "bg-climate-card rounded-xl shadow-lg border border-slate-700 flex flex-col overflow-hidden transition-transform hover:scale-[1.01] duration-300";

        // 1. Header (Region Name)
        let html = `
            <div class="p-4 bg-slate-800 border-b border-slate-700 shadow-inner">
                <h2 class="text-xl font-extrabold text-white text-center tracking-wide">${region.name}</h2>
            </div>
        `;

        // 2. YTD Hero + Trend Badge
        html += `
            <div class="p-8 flex flex-col items-center justify-center border-b border-slate-700 bg-slate-800/30">
                <h3 class="text-slate-400 text-sm uppercase tracking-wider font-semibold mb-2">${currentYearData.year} Records (YTD)</h3>
                <div class="text-7xl font-black text-climate-accent drop-shadow-md">${currentYearData.ytd_records}</div>
                ${buildTrendBadge(data, currentYearData.year)}
            </div>
        `;

        // 3. Recent activity strip (last 90 days)
        html += buildRecentStrip(recentDays, currentYearData.year);

        // 4. Sparkline (full-width SVG bar chart across all years)
        html += `
            <div class="border-b border-slate-700 bg-slate-900/40 py-2">
                ${buildSparkline(data, currentYearData.year, region.id)}
            </div>
        `;

        // 5. Top 5 List (YTD)
        html += `
            <div class="p-5 border-b border-slate-700">
                <h3 class="text-xs font-semibold text-slate-400 mb-3 uppercase tracking-wider">Historical Top 5 (YTD)</h3>
                <ul class="space-y-2 text-sm font-medium">
        `;
        sortedByYtd.forEach(item => {
            const isCurrent = item.year === currentYearData.year;
            html += `
                <li class="flex justify-between items-center bg-slate-800/60 rounded-md px-3 py-2 ${isCurrent ? 'border border-climate-accent/60 shadow-[0_0_10px_rgba(239,68,68,0.2)]' : 'border border-transparent'}">
                    <span class="${isCurrent ? 'font-bold text-white' : 'text-slate-300'}">${item.year}</span>
                    <span class="font-mono ${isCurrent ? 'text-climate-accent font-bold' : 'text-slate-400'}">${item.ytd_records} <span class="text-[10px] text-slate-500 font-sans ml-1">days</span></span>
                </li>
            `;
        });
        html += `</ul></div>`;

        // 6. Historical Table (Scrollable)
        html += `
            <div class="flex flex-col bg-slate-800/20">
                <div class="px-4 py-3 bg-slate-800/90 border-b border-slate-700 flex justify-between text-[11px] font-bold text-slate-400 uppercase tracking-widest">
                    <span class="w-1/3">Year</span>
                    <span class="w-1/3 text-right">YTD</span>
                    <span class="w-1/3 text-right">Full Year</span>
                </div>
                <div class="overflow-y-auto max-h-72 p-2 space-y-1">
        `;

        sortedRecent.forEach(row => {
            const fullYearDisplay = row.full_year_records !== null ? row.full_year_records : '<span class="text-slate-600">-</span>';
            const isCurrent = row.year === currentYearData.year;
            const ytdPct = maxYtd > 0 ? Math.round((row.ytd_records / maxYtd) * 100) : 0;
            const barBg = isCurrent ? 'bg-red-500/40' : 'bg-red-500/20';

            html += `
                <div class="flex justify-between text-sm px-3 py-2 rounded hover:bg-slate-700/60 transition-colors border border-transparent ${isCurrent ? 'bg-slate-800/50 border-climate-accent/30' : ''}">
                    <span class="w-1/3 ${isCurrent ? 'font-bold text-white' : 'text-slate-300'}">${row.year}</span>
                    <div class="w-1/3 relative flex items-center justify-end overflow-hidden">
                        <div class="absolute inset-y-0 right-0 rounded-sm ${barBg}" style="width:${ytdPct}%"></div>
                        <span class="relative font-mono ${isCurrent ? 'text-climate-accent font-bold' : 'text-slate-400'}">${row.ytd_records}</span>
                    </div>
                    <span class="w-1/3 text-right font-mono text-slate-400 ${isCurrent ? 'opacity-50' : ''}">${fullYearDisplay}</span>
                </div>
            `;
        });

        html += `</div></div>`;

        card.innerHTML = html;
        regionsContainer.appendChild(card);
    }

    function hideState(el) { el.classList.add('hidden'); }
    function showState(el) { el.classList.remove('hidden'); }
    function showError(msg) {
        hideState(loadingState);
        hideState(regionsContainer);
        errorMessage.textContent = msg;
        showState(errorState);
    }
});
