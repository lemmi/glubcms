## Testing how script tags are handled

<script>
document.getElementById("demo").innerHTML = "Hello JavaScript!";
</script> 

<script type="text/javascript">alert("XSS");</script>

<span>span-tag</span>
