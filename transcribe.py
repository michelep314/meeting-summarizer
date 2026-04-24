import sys
from faster_whisper import WhisperModel

model = WhisperModel("medium", device="cpu", compute_type="float32")

audio_file = sys.argv[1]  # prende il file audio come argomento

print(f"Trascrivo: {audio_file}")
segments, info = model.transcribe(audio_file, language="it")

print(f"Lingua rilevata: {info.language} (confidenza: {info.language_probability:.0%})")
print("\n--- TRASCRIZIONE ---")
for segment in segments:
    print(f"[{segment.start:.1f}s] {segment.text}")