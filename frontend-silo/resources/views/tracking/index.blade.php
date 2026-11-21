@extends('layouts.shell')

@section('title', 'Riwayat tracking · Silo SCM')

@section('page')
<header class="topbar">
    <div>
        <p class="muted">Batch dari perangkat IoT · masuk otomatis</p>
        <h1>Riwayat tracking</h1>
    </div>
    <div class="top-tools">
        <span class="pill"><span id="track-count">0</span> batch</span>
        <span class="pill" id="track-update">Memuat…</span>
    </div>
</header>

<section class="card">
    <div class="table-wrap">
        <table class="tbl">
            <thead>
                <tr>
                    <th>ID box</th>
                    <th>Varietas</th>
                    <th>Proses</th>
                    <th>Grade</th>
                    <th>Berat</th>
                    <th>Silo</th>
                    <th>Waktu</th>
                    <th>Foto</th>
                    <th class="aksi">Aksi</th>
                </tr>
            </thead>
            <tbody id="track-body">
                <tr><td colspan="9" class="empty">Memuat data…</td></tr>
            </tbody>
        </table>
    </div>
</section>
@endsection

@push('scripts')
<script src="https://cdn.jsdelivr.net/npm/axios@1/dist/axios.min.js"></script>
<script>
window.SILO_CONFIG = {
    apiUrl: @json(config('silo.public_api_url')),
    token: @json($token),
    segarkanUrl: @json(route('silo.token')),
    csrf: @json(csrf_token()),
};
</script>
<script src="{{ asset('js/silo.js') }}"></script>
<script src="{{ asset('js/tracking.js') }}"></script>
@endpush
