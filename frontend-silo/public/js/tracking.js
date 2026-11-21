(function () {
  const S = window.Silo;
  const $ = (id) => document.getElementById(id);
  const LABEL = { lot: 'Lot', frm: 'Kode FRM', var: 'Varietas', prc: 'Proses', dt: 'Tanggal', mc: 'Kadar air (%)', grd: 'Grade', wt: 'Berat (kg)', silo: 'Silo tujuan' };
  const SVG = 'http://www.w3.org/2000/svg';
  const dilihat = new Set();
  let sigTerakhir = null;

  function parseGeo(g) {
    const m = /^(-?\d+(?:\.\d+)?),(-?\d+(?:\.\d+)?)\|(-?\d+)m$/.exec(String(g || ''));
    return m ? { lat: m[1], lng: m[2], alt: m[3] } : null;
  }

  function fotoUrl(kode, nama) {
    return '/tracking/' + encodeURIComponent(kode) + '/foto/' + encodeURIComponent(nama);
  }

  function ikonMata() {
    const svg = document.createElementNS(SVG, 'svg');
    svg.setAttribute('class', 'icon');
    const use = document.createElementNS(SVG, 'use');
    use.setAttribute('href', '#ic-eye');
    svg.append(use);
    return svg;
  }

  // ----- tabel riwayat -----
  function baris(item, baru) {
    const d = item.data || {};
    const foto = item.foto || [];
    const tr = document.createElement('tr');
    if (baru) tr.className = 'row-new';

    const tdFoto = S.el('td');
    if (foto.length) {
      const img = document.createElement('img');
      img.src = fotoUrl(item.batch_id, foto[0]);
      img.alt = 'Foto ' + item.batch_id;
      img.loading = 'lazy';
      img.className = 'thumb';
      tdFoto.append(img);
    } else {
      tdFoto.append(S.el('span', '-', 'muted'));
    }

    const link = S.el('a', 'Lihat detail', 'badge-lihat');
    link.href = '/tracking/' + encodeURIComponent(item.batch_id);
    link.prepend(ikonMata());
    const aksi = S.el('td', null, 'aksi');
    aksi.append(link);

    tr.append(
      S.el('td', item.batch_id, 'mono'),
      S.el('td', d.var || item.material || '-'),
      S.el('td', d.prc || '-'),
      S.el('td', d.grd || '-'),
      S.el('td', d.wt !== undefined ? d.wt + ' kg' : '-'),
      S.el('td', d.silo || '-'),
      S.el('td', S.tanggalJam(item.ts), 'mono'),
      tdFoto,
      aksi
    );
    return tr;
  }

  async function muatDaftar() {
    try {
      const list = (await S.api.get('/tracking', { params: { limit: 200 } })).data.data;
      $('track-count').textContent = list.length;
      $('track-update').textContent = 'Diperbarui ' + S.jamDetik(Date.now() / 1000);

      const sig = list.map((x) => x.batch_id + x.ts + (x.foto || []).length).join('|');
      if (sig === sigTerakhir) return;

      // Batch yang baru masuk disorot sebentar (kecuali saat pertama kali dibuka).
      const baru = sigTerakhir === null ? new Set() : new Set(list.map((x) => x.batch_id).filter((id) => !dilihat.has(id)));
      list.forEach((x) => dilihat.add(x.batch_id));
      sigTerakhir = sig;

      const body = $('track-body');
      body.replaceChildren();
      if (list.length === 0) {
        const td = S.el('td', 'Belum ada batch. Data muncul otomatis saat perangkat memindai barcode.', 'empty');
        td.colSpan = 9;
        const tr = document.createElement('tr');
        tr.append(td);
        body.append(tr);
        return;
      }
      list.forEach((x) => body.append(baris(x, baru.has(x.batch_id))));
    } catch (e) {
      const status = e.response ? e.response.status : 0;
      const pesan = (e.response && e.response.data && e.response.data.pesan) || (status ? 'Error ' + status + ' dari API' : 'API tidak terjangkau');
      $('track-update').textContent = 'Gagal memuat';
      const body = $('track-body');
      body.replaceChildren();
      const td = S.el('td', pesan, 'empty');
      td.colSpan = 9;
      const tr = document.createElement('tr');
      tr.append(td);
      body.append(tr);
    }
  }

  // ----- detail -----
  function baris2(dl, label, nilai, cls) {
    const div = S.el('div');
    div.append(S.el('dt', label), S.el('dd', nilai, cls));
    dl.append(div);
  }

  function tampilFoto(kode, daftar) {
    const grid = $('foto-grid');
    grid.replaceChildren();
    $('foto-count').textContent = daftar.length + ' foto';
    $('foto-kosong').hidden = daftar.length > 0;
    daftar.forEach((nama) => {
      const a = S.el('a', null, 'foto-item');
      a.href = fotoUrl(kode, nama);
      a.target = '_blank';
      a.rel = 'noopener';
      const img = document.createElement('img');
      img.src = fotoUrl(kode, nama);
      img.loading = 'lazy';
      img.alt = nama;
      a.append(img);
      grid.append(a);
    });
  }

  async function muatDetail() {
    const kode = $('detail-root').dataset.kode;
    try {
      const item = (await S.api.get('/tracking/' + encodeURIComponent(kode))).data.data;
      const d = item.data || {};
      $('det-title').textContent = item.batch_id;
      $('det-sub').textContent = (d.var || '-') + ' · ' + (d.prc || '-');
      $('det-time').textContent = 'Dipindai ' + S.tanggalJam(item.ts);

      const dl = $('det-fields');
      dl.replaceChildren();
      baris2(dl, 'ID box', item.batch_id, 'mono');
      Object.keys(LABEL).forEach((k) => { if (d[k] !== undefined) baris2(dl, LABEL[k], String(d[k])); });

      const geo = parseGeo(d.geo);
      if (geo) {
        baris2(dl, 'Koordinat', geo.lat + ', ' + geo.lng, 'mono');
        baris2(dl, 'Ketinggian', geo.alt + ' m');
        const peta = S.el('div');
        const dd = S.el('dd');
        const link = S.el('a', 'Buka di Google Maps');
        link.href = 'https://maps.google.com/?q=' + geo.lat + ',' + geo.lng;
        link.target = '_blank';
        link.rel = 'noopener';
        dd.append(link);
        peta.append(S.el('dt', 'Peta'), dd);
        dl.append(peta);
      }

      $('det-raw').textContent = JSON.stringify(item.data, null, 2);
      tampilFoto(item.batch_id, item.foto || []);
    } catch (e) {
      $('det-status').textContent = e.response && e.response.status === 404
        ? 'Batch tidak ditemukan.'
        : 'Data tidak bisa dimuat (API tidak terjangkau).';
    }
  }

  if ($('track-body')) {
    muatDaftar();
    setInterval(muatDaftar, 4000);
  }
  if ($('detail-root')) {
    muatDetail();
    setInterval(muatDetail, 6000);
  }
})();
