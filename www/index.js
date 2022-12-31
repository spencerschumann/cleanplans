
import * as pdfjsLib from 'pdfjs-dist/webpack';

async function loadGoWASM() {
    const go = new Go();
    let wasm = await WebAssembly.instantiateStreaming(fetch("go.wasm"), go.importObject)
    console.log("*** WebAssembly.instantiateStreaming finished")

    console.log("*** call go.run(inst)")
    // TODO: must ensure that this gets called before calling goCleanPlans later on.
    go.run(wasm.instance);
}
loadGoWASM()

// Get references to the file input, load button, URL input, and load URL button elements
const fileInput = document.getElementById('file-input');

// Define a function to load a PDF file from a local file
function loadPdfFromFile() {
    console.log('Files: ', fileInput.files)

    // TODO: better error handling? Or force the selector to choose a single file?
    if (fileInput.files.length !== 1) {
        console.log('please select one PDF file to process');
        return;
    }

    // Get the selected file from the file input
    const file = fileInput.files[0];
    if (!file) {
        console.error('No file selected');
        return;
    }

    // Read the file as an array buffer
    const fileReader = new FileReader();
    fileReader.onload = function () {
        const arrayBuffer = new Uint8Array(fileReader.result);
        loadPdfFromData(arrayBuffer);
    };
    fileReader.readAsArrayBuffer(file);
    console.log('reading file ' + file + ' ...');
}

// Add event listeners to the load button and load URL button to call the appropriate function
// when they are clicked
fileInput.addEventListener('change', loadPdfFromFile);


async function loadPdfFromData(data) {
    let doc = await pdfjsLib.getDocument(data).promise;
    console.log(`Pages: ${doc.numPages}`);
    let page = await doc.getPage(11);
    console.log(`Page: ${page}`);
    console.log(`  User Units: ${page.userUnit}`);
    console.log(`  View: ${page.view}`);
    console.log(`  Viewport: ${JSON.stringify(page.getViewport())}`);

    // Useful info: await page.getPageLabels()
    // And maybe useful: await page.getOutline()
    // And slight chance of usefulness: await page.getOptionalContentConfig()

    let ops = await page.getOperatorList();
    //console.log(`  Operator list: ${ JSON.stringify(ops) }`);
    let opNames = {}
    for (let name in pdfjsLib.OPS) {
        opNames[pdfjsLib.OPS[name]] = name;
    }
    for (let i = 0; i < ops.fnArray.length; i++) {
        let op = ops.fnArray[i];
        let args = ops.argsArray[i];
        if (op === pdfjsLib.OPS.constructPath) {
            console.log(`   Op: constructPath(minmax = ${JSON.stringify(args[2])})`);
            for (let j = 0; j < args[0].length; j++) {
                let pOp = args[0][j];
                let pArgs = args[1][j]; // WRONG! See https://github.com/mozilla/pdf.js/blob/c791e01bfc280fb6abc476dece21c6a88d2340df/src/display/canvas.js#L1724 for an example of how to interpret the args.
                console.log(`          ${opNames[pOp]}(${JSON.stringify(pArgs)})`);
            }
        } else if (op === pdfjsLib.OPS.paintImageXObject) {
            console.log(`   Op: paintImageXObject ${args[0]}`);
            let obj = page.objs.get(args[0]);
            //console.log(`             Obj: ${ JSON.stringify(obj) }`);
            // obj has  Obj: {"width":1508,"height":2111,"kind":2,"data":"..."}
            console.log(`Type of data: ${obj.data.constructor}`)
            {
                let start = performance.now()
                let sum = 0
                for (let i = 0; i < obj.data.length; i++) {
                    sum += obj.data[i]
                }
                let end = performance.now()
                console.log(`    Time to find average (${sum / obj.data.length}) in JS: ${end - start}`)
            }
            {
                let start = performance.now()
                let result = goCleanPlans(obj.data, obj.width, obj.height, obj.kind);
                let end = performance.now()
                console.log(`called cleanPlans(), result is ${result}`)
                console.log(`     Time to find average in Go: ${end - start}`)
            }
        } else {
            console.log(`   Op: ${opNames[op]}(${JSON.stringify(args)})`);
        }
    }
}
