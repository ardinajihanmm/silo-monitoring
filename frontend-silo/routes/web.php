<?php

use App\Http\Controllers\DashboardController;
use App\Http\Controllers\SiloTokenController;
use App\Http\Controllers\TrackingController;
use Illuminate\Support\Facades\Route;

// Kode box dan nama file: harus diawali huruf/angka dan tanpa "/" (mencegah path traversal).
$kodeRx = '[A-Za-z0-9][A-Za-z0-9._\-]*';

// Ringkasan (dashboard monitoring + kendali), tanpa login.
Route::get('/', [DashboardController::class, 'dashboard'])->name('dashboard');
Route::redirect('/dashboard', '/');

// Token API Go diambil di server. Browser hanya meminta ulang saat token ditolak (401).
Route::post('/silo/token', [SiloTokenController::class, 'segarkan'])->name('silo.token');

// Riwayat tracking: data dan foto masuk otomatis dari perangkat IoT lewat API Go.
Route::get('/tracking', [TrackingController::class, 'index'])->name('tracking.index');
Route::get('/tracking/{kode}', [TrackingController::class, 'show'])->name('tracking.show')->where('kode', $kodeRx);
Route::get('/tracking/{kode}/foto/{file}', [TrackingController::class, 'lihatFoto'])->name('tracking.foto.show')->where(['kode' => $kodeRx, 'file' => $kodeRx]);
