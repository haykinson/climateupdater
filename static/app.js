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
            if (status.last_updated && lastUpdated) {
                const d = new Date(status.last_updated);
                lastUpdated.textContent = `Last Updated: ${d.toLocaleString()}`;
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
                fetch(`/api/records?region=${r.id}`)
                    .then(res => {
                        if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
                        return res.json();
                    })
                    .then(data => ({ region: r, data }))
            );

            return Promise.all(fetchPromises);
        })
        .then(results => {
            hideState(loadingState);
            showState(regionsContainer);

            // Render a card for each successfully fetched region
            results.forEach(result => {
                if (result.data && result.data.length > 0) {
                    renderRegionCard(result.region, result.data);
                }
            });
        })
        .catch(err => {
            console.error("Fetch error:", err);
            // In case the backend is just taking a few seconds to compile historical data (since it does it in the background)
            // let's give the user a clear instruction to retry.
            showError("Failed to fetch dataset. The backend may still be caching historical records. Please refresh the page in a few seconds.");
        });

    function renderRegionCard(region, data) {
        // Extract needed data points
        const currentYearData = data[data.length - 1];
        const sortedByYtd = [...data].sort((a, b) => b.ytd_records - a.ytd_records).slice(0, 5);
        const sortedRecent = [...data].sort((a, b) => b.year - a.year);

        const card = document.createElement('div');
        card.className = "bg-climate-card rounded-xl shadow-lg border border-slate-700 flex flex-col overflow-hidden h-[750px] transition-transform hover:scale-[1.01] duration-300";

        // 1. Header (Region Name)
        let html = `
            <div class="p-4 bg-slate-800 border-b border-slate-700 shadow-inner">
                <h2 class="text-xl font-extrabold text-white text-center tracking-wide">${region.name}</h2>
            </div>
        `;

        // 2. YTD Summary (Giant Highlight)
        html += `
            <div class="p-8 flex flex-col items-center justify-center border-b border-slate-700 bg-slate-800/30">
                <h3 class="text-slate-400 text-sm uppercase tracking-wider font-semibold mb-2">${currentYearData.year} Records (YTD)</h3>
                <div class="text-7xl font-black text-climate-accent drop-shadow-md">${currentYearData.ytd_records}</div>
            </div>
        `;

        // 3. Top 5 List (YTD)
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

        // 4. Historical Table (Scrollable List)
        html += `
            <div class="flex-1 flex flex-col overflow-hidden bg-slate-800/20">
                <div class="px-4 py-3 bg-slate-800/90 border-b border-slate-700 flex justify-between text-[11px] font-bold text-slate-400 uppercase tracking-widest z-10">
                    <span class="w-1/3">Year</span>
                    <span class="w-1/3 text-right">YTD</span>
                    <span class="w-1/3 text-right">Full Year</span>
                </div>
                <div class="overflow-y-auto flex-1 p-2 space-y-1">
        `;

        sortedRecent.forEach(row => {
            const fullYearDisplay = row.full_year_records !== null ? row.full_year_records : '<span class="text-slate-600">-</span>';
            const isCurrent = row.year === currentYearData.year;

            html += `
                <div class="flex justify-between text-sm px-3 py-2 rounded hover:bg-slate-700/60 transition-colors border border-transparent ${isCurrent ? 'bg-slate-800/50 border-climate-accent/30' : ''}">
                    <span class="w-1/3 ${isCurrent ? 'font-bold text-white' : 'text-slate-300'}">${row.year}</span>
                    <span class="w-1/3 text-right font-mono ${isCurrent ? 'text-climate-accent font-bold' : 'text-slate-400'}">${row.ytd_records}</span>
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
