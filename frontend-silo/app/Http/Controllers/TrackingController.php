<?php

namespace App\Http\Controllers;

use Illuminate\Support\Facades\Http;
use Illuminate\View\View;
use Symfony\Component\HttpFoundation\Response;

class TrackingController extends Controller
{
    public function index(): View
    {
        return view('tracking.index', ['token' => SiloTokenController::token()]);
    }

    public function show(string $kode): View
    {
        return view('tracking.show', ['kode' => $kode, 'token' => SiloTokenController::token()]);
    }

    // Foto disimpan di backend Go. Laravel meneruskannya dengan token di sisi server.
    public function lihatFoto(string $kode, string $file): Response
    {
        $respons = Http::withToken((string) SiloTokenController::token())
            ->timeout(10)
            ->get(rtrim(config('silo.api_url'), '/') . '/tracking/' . $kode . '/foto/' . $file);

        abort_unless($respons->successful(), 404);

        return response($respons->body(), 200, [
            'Content-Type' => $respons->header('Content-Type') ?: 'image/jpeg',
            'Cache-Control' => 'public, max-age=3600',
        ]);
    }
}
