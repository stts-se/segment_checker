# Mötesanteckningar 2020-11-27

För det första scenariot:
* Man behöver inte kunna dubbelkolla redan kollade segment
* Max 10.000 segment i ett projekt
* En samtidig användare (hellre det än strul med flera samtidiga användare)
* Vi kan få lite testdata för e-vokaler
* 50ms - 1s långa segment

Action points:
- [x] Ta bort knappar: Bad sample, SKIP, OK, next https://github.com/stts-se/segment_checker/issues/27
- [x] Gör knappar för att flytta segmentgränser? https://github.com/stts-se/segment_checker/issues/28
- [x] Funktion+shortcuts för att flytta segmentgränser långt https://github.com/stts-se/segment_checker/issues/29
- [x] Sätta kontextfönster i param (tillfälligt) https://github.com/stts-se/segment_checker/issues/30
- [ ] Visualisera progress? https://github.com/stts-se/segment_checker/issues/31
- [ ] Nåt slags badge? https://github.com/stts-se/segment_checker/issues/32
- [x] ~~Låsa appen för samtidiga användare? https://github.com/stts-se/segment_checker/issues/33~~

Funderingar:
* Behöver vi ändå implementera en enkel dubbelkoll för att kunna testa att verktyget funkar som det ska? (utan möjlighet att spara) https://github.com/stts-se/segment_checker/issues/23
* Läsa in käll- och annotationdata i minnet istf att läsa från disk https://github.com/stts-se/segment_checker/issues/26

Issues för MVP: https://github.com/stts-se/segment_checker/labels/MVP

